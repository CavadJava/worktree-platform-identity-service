import { useState } from 'react';
import { Button, Form, Input, Modal, Table } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createProject, listProjects } from '../api/projects';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';
import { message } from 'antd';

interface ProjectFormValues {
  name: string;
}

export function ProjectsPage() {
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm<ProjectFormValues>();

  const { data: projects, isLoading, isError, error } = useQuery({ queryKey: ['projects'], queryFn: () => listProjects() });
  useQueryErrorToast(isError, error);

  const createMutation = useMutation({
    mutationFn: (values: ProjectFormValues) => createProject(values.name),
    onSuccess: () => {
      message.success('Layihə yaradıldı');
      setModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'ID', dataIndex: 'id' },
    { title: 'Yaradılma tarixi', dataIndex: 'created_at' },
  ];

  return (
    <>
      <Button type="primary" onClick={() => setModalOpen(true)} style={{ marginBottom: 16 }}>
        Yeni layihə
      </Button>
      <Table rowKey="id" loading={isLoading} dataSource={projects} columns={columns} />
      <Modal
        title="Yeni layihə"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="name" label="Ad" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
