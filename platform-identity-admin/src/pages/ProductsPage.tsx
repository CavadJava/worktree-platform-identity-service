import { useState } from 'react';
import { Button, Form, Input, List, Modal, Select, Space, Switch, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Product, Subproject } from '../api/types';
import {
  addSubproject,
  createProduct,
  createProductUser,
  listProducts,
  listSubprojects,
  removeSubproject,
  setSubscription,
  updateProductProfile,
} from '../api/products';
import { listAllUsers } from '../api/users';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

interface ProductFormValues {
  name: string;
}

interface SubprojectFormValues {
  name: string;
  description: string;
}

interface ProductUserFormValues {
  name: string;
  username: string;
  email: string;
  password: string;
}

export function ProductsPage() {
  const queryClient = useQueryClient();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [subModalProduct, setSubModalProduct] = useState<Product | null>(null);
  const [subUserId, setSubUserId] = useState<string | null>(null);
  const [subscripted, setSubscripted] = useState(false);
  const [renewed, setRenewed] = useState(false);
  const [subNotes, setSubNotes] = useState('');
  const [profileModalProduct, setProfileModalProduct] = useState<Product | null>(null);
  const [userModalProduct, setUserModalProduct] = useState<Product | null>(null);
  const [form] = Form.useForm<ProductFormValues>();
  const [profileForm] = Form.useForm<{ description: string; techStack: string }>();
  const [subprojectForm] = Form.useForm<SubprojectFormValues>();
  const [productUserForm] = Form.useForm<ProductUserFormValues>();

  const { data: products, isLoading, isError, error } = useQuery({ queryKey: ['products'], queryFn: () => listProducts() });
  useQueryErrorToast(isError, error);

  const { data: allUsers } = useQuery({ queryKey: ['all-users-for-sub'], queryFn: () => listAllUsers(), enabled: !!subModalProduct });

  const { data: subprojects } = useQuery({
    queryKey: ['product-subprojects', profileModalProduct?.id],
    queryFn: () => listSubprojects(profileModalProduct!.id),
    enabled: !!profileModalProduct,
  });

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
    mutationFn: () => setSubscription(subUserId!, subModalProduct!.id, subscripted, renewed, subNotes),
    onSuccess: () => {
      message.success('Subscription yeniləndi');
      setSubModalProduct(null);
      setSubUserId(null);
      setSubscripted(false);
      setRenewed(false);
      setSubNotes('');
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const profileMutation = useMutation({
    mutationFn: (values: { description: string; techStack: string }) =>
      updateProductProfile(profileModalProduct!.id, values.description, values.techStack),
    onSuccess: () => {
      message.success('Profil yeniləndi');
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const addSubprojectMutation = useMutation({
    mutationFn: (values: SubprojectFormValues) =>
      addSubproject(profileModalProduct!.id, values.name, values.description),
    onSuccess: () => {
      subprojectForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['product-subprojects', profileModalProduct?.id] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const removeSubprojectMutation = useMutation({
    mutationFn: (subId: string) => removeSubproject(profileModalProduct!.id, subId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['product-subprojects', profileModalProduct?.id] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const createProductUserMutation = useMutation({
    mutationFn: (values: ProductUserFormValues) =>
      createProductUser(userModalProduct!.id, values.name, values.username, values.email, values.password),
    onSuccess: () => {
      message.success('İstifadəçi yaradıldı və məhsula abunə edildi');
      setUserModalProduct(null);
      productUserForm.resetFields();
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const openProfileModal = (product: Product) => {
    setProfileModalProduct(product);
    profileForm.setFieldsValue({ description: product.description, techStack: product.tech_stack });
  };

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'ID', dataIndex: 'id' },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: Product) => (
        <Space>
          <Button size="small" onClick={() => openProfileModal(record)}>
            Profil
          </Button>
          <Button size="small" onClick={() => setSubModalProduct(record)}>
            Subscription idarə et
          </Button>
          <Button size="small" onClick={() => setUserModalProduct(record)}>
            Yeni istifadəçi
          </Button>
        </Space>
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
          <Input.TextArea
            rows={3}
            placeholder="Müştəri haqqında qeyd"
            value={subNotes}
            onChange={(e) => setSubNotes(e.target.value)}
          />
        </Space>
      </Modal>
      <Modal
        title={profileModalProduct ? `${profileModalProduct.name} — Profil` : ''}
        open={!!profileModalProduct}
        onCancel={() => setProfileModalProduct(null)}
        footer={null}
        width={640}
      >
        <Form
          form={profileForm}
          layout="vertical"
          onFinish={(values) => profileMutation.mutate(values)}
        >
          <Form.Item name="description" label="Təsvir">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="techStack" label="Tech stack (vergüllə ayrılmış, məs. React, Go, Postgres)">
            <Input />
          </Form.Item>
          <Form.Item shouldUpdate>
            {() => {
              const raw = profileForm.getFieldValue('techStack') as string | undefined;
              const tags = (raw ?? '').split(',').map((t) => t.trim()).filter(Boolean);
              return (
                <Space wrap>
                  {tags.map((tag) => (
                    <Tag key={tag}>{tag}</Tag>
                  ))}
                </Space>
              );
            }}
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={profileMutation.isPending}>
            Profili yadda saxla
          </Button>
        </Form>

        <div style={{ marginTop: 24 }}>
          <h4>Daxili layihələr</h4>
          <List
            size="small"
            dataSource={subprojects ?? []}
            renderItem={(sub: Subproject) => (
              <List.Item
                actions={[
                  <Button
                    key="remove"
                    size="small"
                    danger
                    onClick={() => removeSubprojectMutation.mutate(sub.id)}
                  >
                    Sil
                  </Button>,
                ]}
              >
                <List.Item.Meta title={sub.name} description={sub.description} />
              </List.Item>
            )}
          />
          <Form
            form={subprojectForm}
            layout="inline"
            style={{ marginTop: 12 }}
            onFinish={(values) => addSubprojectMutation.mutate(values)}
          >
            <Form.Item name="name" rules={[{ required: true, message: 'Ad tələb olunur' }]}>
              <Input placeholder="Ad (məs. auth-service)" />
            </Form.Item>
            <Form.Item name="description">
              <Input placeholder="Təsvir" />
            </Form.Item>
            <Form.Item>
              <Button htmlType="submit" loading={addSubprojectMutation.isPending}>
                Əlavə et
              </Button>
            </Form.Item>
          </Form>
        </div>
      </Modal>
      <Modal
        title={userModalProduct ? `${userModalProduct.name} — Yeni istifadəçi` : ''}
        open={!!userModalProduct}
        onCancel={() => setUserModalProduct(null)}
        onOk={() => productUserForm.submit()}
        confirmLoading={createProductUserMutation.isPending}
      >
        <Form
          form={productUserForm}
          layout="vertical"
          onFinish={(values) => createProductUserMutation.mutate(values)}
        >
          <Form.Item name="name" label="Ad" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="username" label="İstifadəçi adı" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="Parol" rules={[{ required: true, min: 6 }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
