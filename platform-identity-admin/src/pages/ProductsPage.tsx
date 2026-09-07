import { useState } from 'react';
import { Button, Form, Input, List, Modal, Select, Space, Switch, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { BasicUser, Product, ProductAdminRequest, ProductBrowse, Subproject } from '../api/types';
import {
  addSubproject,
  createProduct,
  createProductUser,
  decideAdminRequest,
  listAdminRequests,
  listBrowseProducts,
  listMyProducts,
  listSubprojects,
  promoteProductAdmin,
  removeSubproject,
  requestProductAdmin,
  setSubscription,
  updateProductProfile,
} from '../api/products';
import { listBasicUsers } from '../api/users';
import { useAuth } from '../auth/AuthContext';
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
  systemRole: string;
}

export function ProductsPage() {
  const queryClient = useQueryClient();
  const { claims } = useAuth();
  const isSuperadmin = claims?.system_role === 'superadmin';
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [subModalProduct, setSubModalProduct] = useState<Product | null>(null);
  const [subUserId, setSubUserId] = useState<string | null>(null);
  const [subscripted, setSubscripted] = useState(false);
  const [renewed, setRenewed] = useState(false);
  const [subNotes, setSubNotes] = useState('');
  const [profileModalProduct, setProfileModalProduct] = useState<Product | null>(null);
  const [userModalProduct, setUserModalProduct] = useState<Product | null>(null);
  const [requestsModalProduct, setRequestsModalProduct] = useState<Product | null>(null);
  const [promoteModalProduct, setPromoteModalProduct] = useState<Product | null>(null);
  const [promoteUserId, setPromoteUserId] = useState<string | null>(null);
  const [form] = Form.useForm<ProductFormValues>();
  const [profileForm] = Form.useForm<{ description: string; techStack: string }>();
  const [subprojectForm] = Form.useForm<SubprojectFormValues>();
  const [productUserForm] = Form.useForm<ProductUserFormValues>();

  const { data: products, isLoading, isError, error } = useQuery({ queryKey: ['my-products'], queryFn: () => listMyProducts() });
  useQueryErrorToast(isError, error);

  const { data: browseProducts } = useQuery({ queryKey: ['browse-products'], queryFn: () => listBrowseProducts() });

  const { data: basicUsers } = useQuery({ queryKey: ['basic-users'], queryFn: () => listBasicUsers(), enabled: !!subModalProduct || !!promoteModalProduct });

  const { data: subprojects } = useQuery({
    queryKey: ['product-subprojects', profileModalProduct?.id],
    queryFn: () => listSubprojects(profileModalProduct!.id),
    enabled: !!profileModalProduct,
  });

  const { data: adminRequests } = useQuery({
    queryKey: ['admin-requests', requestsModalProduct?.id],
    queryFn: () => listAdminRequests(requestsModalProduct!.id),
    enabled: !!requestsModalProduct,
  });

  const createMutation = useMutation({
    mutationFn: (values: ProductFormValues) => createProduct(values.name),
    onSuccess: () => {
      message.success('Product yaradıldı');
      setCreateModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['my-products'] });
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
      queryClient.invalidateQueries({ queryKey: ['my-products'] });
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
      createProductUser(userModalProduct!.id, values.name, values.username, values.email, values.password, values.systemRole),
    onSuccess: () => {
      message.success('İstifadəçi yaradıldı və məhsula abunə edildi');
      setUserModalProduct(null);
      productUserForm.resetFields();
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const requestAdminMutation = useMutation({
    mutationFn: (subjectUserId: string) => requestProductAdmin(promoteModalProduct!.id, subjectUserId),
    onSuccess: () => {
      message.success('Sorğu göndərildi, superadmin təsdiqini gözləyir');
      setPromoteModalProduct(null);
      setPromoteUserId(null);
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const promoteDirectlyMutation = useMutation({
    mutationFn: (subjectUserId: string) => promoteProductAdmin(promoteModalProduct!.id, subjectUserId),
    onSuccess: () => {
      message.success('İstifadəçi admin təyin olundu');
      setPromoteModalProduct(null);
      setPromoteUserId(null);
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const decideRequestMutation = useMutation({
    mutationFn: ({ requestId, approve }: { requestId: string; approve: boolean }) =>
      decideAdminRequest(requestsModalProduct!.id, requestId, approve),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-requests', requestsModalProduct?.id] });
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
        <Space wrap>
          <Button size="small" onClick={() => openProfileModal(record)}>
            Profil
          </Button>
          <Button size="small" onClick={() => setSubModalProduct(record)}>
            Subscription idarə et
          </Button>
          <Button size="small" onClick={() => setUserModalProduct(record)}>
            Yeni istifadəçi
          </Button>
          <Button size="small" onClick={() => setRequestsModalProduct(record)}>
            Admin sorğuları
          </Button>
          <Button size="small" onClick={() => setPromoteModalProduct(record)}>
            Admin təyin et
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
      {!isSuperadmin && (products?.length ?? 0) === 0 && (
        <div style={{ marginTop: 24 }}>
          <h4>Bütün productlar</h4>
          <List
            size="small"
            dataSource={browseProducts ?? []}
            renderItem={(p: ProductBrowse) => (
              <List.Item
                actions={[
                  <Button
                    key="request"
                    size="small"
                    onClick={() => {
                      setPromoteModalProduct(p as Product);
                      setPromoteUserId(null);
                    }}
                  >
                    Sorğu göndər
                  </Button>,
                ]}
              >
                <List.Item.Meta title={p.name} />
              </List.Item>
            )}
          />
        </div>
      )}
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
            options={basicUsers?.map((u: BasicUser) => ({ label: `${u.name} (${u.username})`, value: u.id }))}
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
          <Form.Item name="systemRole" label="Sistem rolu" initialValue="user" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'user', value: 'user' },
                { label: 'admin', value: 'admin' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={requestsModalProduct ? `${requestsModalProduct.name} — Admin sorğuları` : ''}
        open={!!requestsModalProduct}
        onCancel={() => setRequestsModalProduct(null)}
        footer={null}
      >
        <List
          size="small"
          dataSource={adminRequests ?? []}
          locale={{ emptyText: 'Gözləyən sorğu yoxdur' }}
          renderItem={(req: ProductAdminRequest) => (
            <List.Item
              actions={
                isSuperadmin
                  ? [
                      <Button key="approve" size="small" type="primary" onClick={() => decideRequestMutation.mutate({ requestId: req.id, approve: true })}>
                        Approve
                      </Button>,
                      <Button key="reject" size="small" danger onClick={() => decideRequestMutation.mutate({ requestId: req.id, approve: false })}>
                        Reject
                      </Button>,
                    ]
                  : []
              }
            >
              <List.Item.Meta title={req.subject_user_id} description={`Status: ${req.status}, göndərən: ${req.requested_by_user_id}`} />
            </List.Item>
          )}
        />
      </Modal>
      <Modal
        title={promoteModalProduct ? `${promoteModalProduct.name} — Admin təyin et` : ''}
        open={!!promoteModalProduct}
        onCancel={() => {
          setPromoteModalProduct(null);
          setPromoteUserId(null);
        }}
        onOk={() => (isSuperadmin ? promoteDirectlyMutation.mutate(promoteUserId!) : requestAdminMutation.mutate(promoteUserId!))}
        okText={isSuperadmin ? 'Birbaşa təyin et' : 'Sorğu göndər'}
        confirmLoading={isSuperadmin ? promoteDirectlyMutation.isPending : requestAdminMutation.isPending}
        okButtonProps={{ disabled: !promoteUserId }}
      >
        <Select
          style={{ width: '100%' }}
          placeholder="İstifadəçi seç"
          value={promoteUserId ?? undefined}
          onChange={setPromoteUserId}
          options={basicUsers?.map((u: BasicUser) => ({ label: `${u.name} (${u.username}) — ${u.system_role}`, value: u.id }))}
        />
      </Modal>
    </>
  );
}
