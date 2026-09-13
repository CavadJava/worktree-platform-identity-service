import { useState } from 'react';
import { Button, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import type { Shop } from '../api/types';
import { createShop, listShops, updateShopProfile } from '../api/shops';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

interface ShopFormValues {
  name: string;
  shop_type: string;
}

interface ShopProfileFormValues {
  contact_email: string;
  contact_phone: string;
  address: string;
  work_hours: string;
}

export function ShopsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [profileModalShop, setProfileModalShop] = useState<Shop | null>(null);
  const [form] = Form.useForm<ShopFormValues>();
  const [profileForm] = Form.useForm<ShopProfileFormValues>();

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

  const profileMutation = useMutation({
    mutationFn: (values: ShopProfileFormValues) =>
      updateShopProfile(
        profileModalShop!.id,
        values.contact_email,
        values.contact_phone,
        values.address,
        values.work_hours
      ),
    onSuccess: () => {
      message.success('Satıcı məlumatları yeniləndi');
      setProfileModalShop(null);
      queryClient.invalidateQueries({ queryKey: ['shops'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const openProfileModal = (shop: Shop) => {
    setProfileModalShop(shop);
    profileForm.setFieldsValue({
      contact_email: shop.contact_email,
      contact_phone: shop.contact_phone,
      address: shop.address,
      work_hours: shop.work_hours,
    });
  };

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    {
      title: 'Növ',
      dataIndex: 'shop_type',
      render: (shopType: string) =>
        shopType === 'foreign' ? <Tag color="blue">Xarici</Tag> : <Tag color="green">Yerli</Tag>,
    },
    {
      title: 'Əlaqə',
      render: (_: unknown, record: Shop) => (
        <Space direction="vertical" size={0}>
          {record.contact_email && <span>{record.contact_email}</span>}
          {record.contact_phone && <span>{record.contact_phone}</span>}
          {!record.contact_email && !record.contact_phone && <span style={{ color: '#999' }}>—</span>}
        </Space>
      ),
    },
    {
      title: 'Ünvan',
      dataIndex: 'address',
      render: (address: string) => address || <span style={{ color: '#999' }}>—</span>,
    },
    {
      title: 'İş saatları',
      dataIndex: 'work_hours',
      render: (workHours: string) => workHours || <span style={{ color: '#999' }}>—</span>,
    },
    { title: 'Yaradılma tarixi', dataIndex: 'created_at' },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: Shop) => (
        <Space>
          <Button size="small" onClick={() => openProfileModal(record)}>
            Profil
          </Button>
          <Button size="small" onClick={() => navigate(`/shops/${record.id}/members`)}>
            Üzvlər
          </Button>
        </Space>
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
      <Modal
        title={profileModalShop ? `${profileModalShop.name} — Profil` : ''}
        open={!!profileModalShop}
        onCancel={() => setProfileModalShop(null)}
        onOk={() => profileForm.submit()}
        confirmLoading={profileMutation.isPending}
      >
        <Form form={profileForm} layout="vertical" onFinish={(values) => profileMutation.mutate(values)}>
          <Form.Item name="contact_email" label="Email">
            <Input />
          </Form.Item>
          <Form.Item name="contact_phone" label="Telefon">
            <Input />
          </Form.Item>
          <Form.Item name="address" label="Ünvan">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="work_hours" label="İş saatları">
            <Input placeholder="məs. Hər gün 09:00–18:00" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
