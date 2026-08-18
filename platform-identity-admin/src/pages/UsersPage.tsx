import { useState } from 'react';
import { Button, Select, Space, Table, Typography, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { User } from '../api/types';
import { listProjects } from '../api/projects';
import { listAllUsers, listUsersByProject, setUserRole } from '../api/users';
import { useAuth } from '../auth/AuthContext';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

export function UsersPage() {
  const { claims } = useAuth();
  const isSuperadmin = claims?.role === 'superadmin';
  const queryClient = useQueryClient();
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);

  const { data: projects, isLoading: projectsLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => listProjects(),
    enabled: !isSuperadmin,
  });

  const effectiveProjectId = selectedProjectId ?? projects?.[0]?.id ?? null;
  const effectiveProjectName = projects?.find((p) => p.id === effectiveProjectId)?.name ?? null;

  const {
    data: scopedUsers,
    isLoading: scopedUsersLoading,
    isError: scopedIsError,
    error: scopedError,
  } = useQuery({
    queryKey: ['project-users', effectiveProjectId],
    queryFn: () => listUsersByProject(effectiveProjectId!),
    enabled: !isSuperadmin && !!effectiveProjectId,
  });

  const {
    data: allUsers,
    isLoading: allUsersLoading,
    isError: allIsError,
    error: allError,
  } = useQuery({
    queryKey: ['all-users'],
    queryFn: () => listAllUsers(),
    enabled: isSuperadmin,
  });

  useQueryErrorToast(isSuperadmin ? allIsError : scopedIsError, isSuperadmin ? allError : scopedError);

  const users = isSuperadmin ? allUsers : scopedUsers;
  const usersLoading = isSuperadmin ? allUsersLoading : scopedUsersLoading;

  const setRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) => setUserRole(userId, role),
    onSuccess: () => {
      message.success('Rol dəyişdirildi');
      if (isSuperadmin) {
        queryClient.invalidateQueries({ queryKey: ['all-users'] });
      } else {
        queryClient.invalidateQueries({ queryKey: ['project-users', effectiveProjectId] });
      }
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'Username', dataIndex: 'username' },
    { title: 'Email', dataIndex: 'email' },
    { title: 'Rol', dataIndex: 'role' },
    ...(isSuperadmin ? [{ title: 'Layihə', dataIndex: 'project_name' }] : []),
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: User) => (
        <Space>
          <Button
            size="small"
            disabled={record.role === 'admin'}
            onClick={() => setRoleMutation.mutate({ userId: record.id, role: 'admin' })}
          >
            Admin et
          </Button>
          <Button
            size="small"
            disabled={record.role === 'user'}
            onClick={() => setRoleMutation.mutate({ userId: record.id, role: 'user' })}
          >
            User et
          </Button>
        </Space>
      ),
    },
  ];

  if (isSuperadmin) {
    return <Table rowKey="id" loading={usersLoading} dataSource={users} columns={columns} />;
  }

  return (
    <>
      <Select
        style={{ width: 320, marginBottom: 16 }}
        placeholder="Layihə seç"
        loading={projectsLoading}
        value={effectiveProjectId ?? undefined}
        onChange={(value) => setSelectedProjectId(value)}
        options={projects?.map((p) => ({ label: p.name, value: p.id }))}
      />
      {effectiveProjectName && (
        <Typography.Title level={5} style={{ marginBottom: 12 }}>
          {effectiveProjectName}
        </Typography.Title>
      )}
      <Table rowKey="id" loading={usersLoading} dataSource={users} columns={columns} />
    </>
  );
}
