import { useState } from 'react';
import { Button, Modal, Select, Space, Table, Typography, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import type { Member } from '../api/types';
import { getShop } from '../api/shops';
import { listShopRoles } from '../api/systemRoles';
import { addShopMember, listAllUsers, listShopMembers, setMemberRole } from '../api/users';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

export function ShopMembersPage() {
  const { id: shopId } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedShopRole, setSelectedShopRole] = useState<string | null>(null);

  const { data: shop } = useQuery({ queryKey: ['shop', shopId], queryFn: () => getShop(shopId!), enabled: !!shopId });

  const {
    data: members,
    isLoading: membersLoading,
    isError,
    error,
  } = useQuery({ queryKey: ['shop-members', shopId], queryFn: () => listShopMembers(shopId!), enabled: !!shopId });
  useQueryErrorToast(isError, error);

  const { data: allUsers } = useQuery({ queryKey: ['all-users-for-add'], queryFn: () => listAllUsers(), enabled: modalOpen });
  const { data: shopRoles } = useQuery({ queryKey: ['shop-roles'], queryFn: () => listShopRoles(), enabled: modalOpen });

  const addMutation = useMutation({
    mutationFn: () => addShopMember(shopId!, selectedUserId!, selectedShopRole!),
    onSuccess: () => {
      message.success('Üzv əlavə edildi');
      setModalOpen(false);
      setSelectedUserId(null);
      setSelectedShopRole(null);
      queryClient.invalidateQueries({ queryKey: ['shop-members', shopId] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const setRoleMutation = useMutation({
    mutationFn: ({ userId, shopRole }: { userId: string; shopRole: string }) => setMemberRole(shopId!, userId, shopRole),
    onSuccess: () => {
      message.success('Rol dəyişdirildi');
      queryClient.invalidateQueries({ queryKey: ['shop-members', shopId] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'User ID', dataIndex: 'user_id' },
    { title: 'Shop rolu', dataIndex: 'shop_role' },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: Member) => (
        <Space>
          <Button
            size="small"
            disabled={record.shop_role === 'shop-admin'}
            onClick={() => setRoleMutation.mutate({ userId: record.user_id, shopRole: 'shop-admin' })}
          >
            Shop-admin et
          </Button>
          <Button
            size="small"
            disabled={record.shop_role === 'shop-user'}
            onClick={() => setRoleMutation.mutate({ userId: record.user_id, shopRole: 'shop-user' })}
          >
            Shop-user et
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      {shop && (
        <Typography.Title level={5} style={{ marginBottom: 12 }}>
          {shop.name} — Üzvlər
        </Typography.Title>
      )}
      <Button type="primary" onClick={() => setModalOpen(true)} style={{ marginBottom: 16 }}>
        Üzv əlavə et
      </Button>
      <Table rowKey="id" loading={membersLoading} dataSource={members} columns={columns} />
      <Modal
        title="Üzv əlavə et"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => addMutation.mutate()}
        confirmLoading={addMutation.isPending}
        okButtonProps={{ disabled: !selectedUserId || !selectedShopRole }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Select
            style={{ width: '100%' }}
            placeholder="İstifadəçi seç"
            value={selectedUserId ?? undefined}
            onChange={setSelectedUserId}
            options={allUsers?.map((u) => ({ label: `${u.name} (${u.username})`, value: u.id }))}
          />
          <Select
            style={{ width: '100%' }}
            placeholder="Rol seç"
            value={selectedShopRole ?? undefined}
            onChange={setSelectedShopRole}
            options={shopRoles?.map((r) => ({ label: r.name, value: r.name }))}
          />
        </Space>
      </Modal>
    </>
  );
}
