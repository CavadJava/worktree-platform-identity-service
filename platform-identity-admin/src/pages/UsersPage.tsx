import { useState } from 'react';
import { Button, Select, Space, Table, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { User } from '../api/types';
import { listProjects } from '../api/projects';
import { listUsersByProject, setUserRole } from '../api/users';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

export function UsersPage() {
  const queryClient = useQueryClient();
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);

  const { data: projects, isLoading: projectsLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => listProjects(),
  });

  const effectiveProjectId = selectedProjectId ?? projects?.[0]?.id ?? null;

  const {
    data: users,
    isLoading: usersLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ['project-users', effectiveProjectId],
    queryFn: () => listUsersByProject(effectiveProjectId!),
    enabled: !!effectiveProjectId,
  });
  useQueryErrorToast(isError, error);

  const setRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) => setUserRole(userId, role),
    onSuccess: () => {
      message.success('Rol dəyişdirildi');
      queryClient.invalidateQueries({ queryKey: ['project-users', effectiveProjectId] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'Username', dataIndex: 'username' },
    { title: 'Email', dataIndex: 'email' },
    { title: 'Rol', dataIndex: 'role' },
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
      <Table rowKey="id" loading={usersLoading} dataSource={users} columns={columns} />
    </>
  );
}
