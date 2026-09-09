import { useState } from 'react';
import { Button, Form, Input, Modal, Select, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import type { Shop } from '../api/types';
import { createShop, listShops } from '../api/shops';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

interface ShopFormValues {
  name: string;
  shop_type: string;
}

export function ShopsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm<ShopFormValues>();

  const { data: shops, isLoading, isError, error } = useQuery({ queryKey: ['shops'], queryFn: () => listShops() });
  useQueryErrorToast(isError, error);

  const createMutation = useMutation({
    mutationFn: (values: ShopFormValues) => createShop(values.name, values.shop_type),
    onSuccess: () => {
      message.success('Shop yaradıldı');
      setModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['shops'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'ID', dataIndex: 'id' },
    {
      title: 'Növ',
      dataIndex: 'shop_type',
      render: (shopType: string) =>
        shopType === 'foreign' ? <Tag color="blue">Xarici</Tag> : <Tag color="green">Yerli</Tag>,
    },
    { title: 'Yaradılma tarixi', dataIndex: 'created_at' },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: Shop) => (
        <Button size="small" onClick={() => navigate(`/shops/${record.id}/members`)}>
          Üzvlər
        </Button>
      ),
    },
  ];

  return (
    <>
      <Button type="primary" onClick={() => setModalOpen(true)} style={{ marginBottom: 16 }}>
        Yeni shop
      </Button>
      <Table rowKey="id" loading={isLoading} dataSource={shops} columns={columns} />
      <Modal
        title="Yeni shop"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="name" label="Ad" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="shop_type" label="Növ" rules={[{ required: true }]}>
            <Select
              placeholder="Növ seç"
              options={[
                { label: 'Xarici', value: 'foreign' },
                { label: 'Yerli', value: 'local' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
