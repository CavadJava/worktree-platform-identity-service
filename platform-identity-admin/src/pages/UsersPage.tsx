import { useState } from 'react';
import { Button, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { User } from '../api/types';
import { createUser, listAllUsers, setStatus, setSystemRole } from '../api/users';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

interface CreateUserFormValues {
  name: string;
  username: string;
  email: string;
  password: string;
  system_role: string;
}

export function UsersPage() {
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm<CreateUserFormValues>();

  const { data: users, isLoading, isError, error } = useQuery({ queryKey: ['all-users'], queryFn: () => listAllUsers() });
  useQueryErrorToast(isError, error);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['all-users'] });

  const createMutation = useMutation({
    mutationFn: (values: CreateUserFormValues) =>
      createUser(values.name, values.username, values.email, values.password, values.system_role),
    onSuccess: () => {
      message.success('İstifadəçi yaradıldı');
      setModalOpen(false);
      form.resetFields();
      invalidate();
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const roleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) => setSystemRole(userId, role),
    onSuccess: () => {
      message.success('Sistem rolu dəyişdirildi');
      invalidate();
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const statusMutation = useMutation({
    mutationFn: ({ userId, status }: { userId: string; status: string }) => setStatus(userId, status),
    onSuccess: () => {
      message.success('Status dəyişdirildi');
      invalidate();
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'Username', dataIndex: 'username' },
    { title: 'Email', dataIndex: 'email' },
    { title: 'Sistem rolu', dataIndex: 'system_role' },
    {
      title: 'Status',
      dataIndex: 'status',
      render: (status: string) => (status === 'ACTIVE' ? <Tag color="green">ACTIVE</Tag> : <Tag color="red">IN_ACTIVE</Tag>),
    },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: User) => (
        <Space direction="vertical">
          <Space>
            <Button
              size="small"
              disabled={record.system_role === 'superadmin'}
              onClick={() => roleMutation.mutate({ userId: record.id, role: 'superadmin' })}
            >
              Superadmin et
            </Button>
            <Button
              size="small"
              disabled={record.system_role === 'admin'}
              onClick={() => roleMutation.mutate({ userId: record.id, role: 'admin' })}
            >
              Admin et
            </Button>
            <Button
              size="small"
              disabled={record.system_role === 'user'}
              onClick={() => roleMutation.mutate({ userId: record.id, role: 'user' })}
            >
              User et
            </Button>
          </Space>
          <Button
            size="small"
            danger={record.status === 'ACTIVE'}
            onClick={() =>
              statusMutation.mutate({ userId: record.id, status: record.status === 'ACTIVE' ? 'IN_ACTIVE' : 'ACTIVE' })
            }
          >
            {record.status === 'ACTIVE' ? 'Deaktiv et' : 'Aktiv et'}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Button type="primary" onClick={() => setModalOpen(true)} style={{ marginBottom: 16 }}>
        Yeni istifadəçi
      </Button>
      <Table rowKey="id" loading={isLoading} dataSource={users} columns={columns} />
      <Modal
        title="Yeni istifadəçi"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="name" label="Ad" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="username" label="Username" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="Şifrə" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="system_role" label="Sistem rolu" rules={[{ required: true }]} initialValue="user">
            <Select
              options={[
                { label: 'user', value: 'user' },
                { label: 'admin', value: 'admin' },
                { label: 'superadmin', value: 'superadmin' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
