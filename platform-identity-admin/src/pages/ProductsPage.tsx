import { useState } from 'react';
import { Button, Form, Input, Modal, Select, Space, Switch, Table, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Product } from '../api/types';
import { createProduct, listProducts, setSubscription } from '../api/products';
import { listAllUsers } from '../api/users';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

interface ProductFormValues {
  name: string;
}

export function ProductsPage() {
  const queryClient = useQueryClient();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [subModalProduct, setSubModalProduct] = useState<Product | null>(null);
  const [subUserId, setSubUserId] = useState<string | null>(null);
  const [subscripted, setSubscripted] = useState(false);
  const [renewed, setRenewed] = useState(false);
  const [form] = Form.useForm<ProductFormValues>();

  const { data: products, isLoading, isError, error } = useQuery({ queryKey: ['products'], queryFn: () => listProducts() });
  useQueryErrorToast(isError, error);

  const { data: allUsers } = useQuery({ queryKey: ['all-users-for-sub'], queryFn: () => listAllUsers(), enabled: !!subModalProduct });

  const createMutation = useMutation({
    mutationFn: (values: ProductFormValues) => createProduct(values.name),
    onSuccess: () => {
      message.success('Product yaradıldı');
      setCreateModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const subMutation = useMutation({
    mutationFn: () => setSubscription(subUserId!, subModalProduct!.id, subscripted, renewed),
    onSuccess: () => {
      message.success('Subscription yeniləndi');
      setSubModalProduct(null);
      setSubUserId(null);
      setSubscripted(false);
      setRenewed(false);
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'ID', dataIndex: 'id' },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: Product) => (
        <Button size="small" onClick={() => setSubModalProduct(record)}>
          Subscription idarə et
        </Button>
      ),
    },
  ];

  return (
    <>
      <Button type="primary" onClick={() => setCreateModalOpen(true)} style={{ marginBottom: 16 }}>
        Yeni product
      </Button>
      <Table rowKey="id" loading={isLoading} dataSource={products} columns={columns} />
      <Modal
        title="Yeni product"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="name" label="Ad" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={subModalProduct ? `${subModalProduct.name} — Subscription` : ''}
        open={!!subModalProduct}
        onCancel={() => setSubModalProduct(null)}
        onOk={() => subMutation.mutate()}
        confirmLoading={subMutation.isPending}
        okButtonProps={{ disabled: !subUserId }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Select
            style={{ width: '100%' }}
            placeholder="İstifadəçi seç"
            value={subUserId ?? undefined}
            onChange={setSubUserId}
            options={allUsers?.map((u) => ({ label: `${u.name} (${u.username})`, value: u.id }))}
          />
          <Space>
            <span>Subscripted:</span>
            <Switch checked={subscripted} onChange={setSubscripted} />
          </Space>
          <Space>
            <span>Renewed:</span>
            <Switch checked={renewed} onChange={setRenewed} />
          </Space>
        </Space>
      </Modal>
    </>
  );
}
