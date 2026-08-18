import { Button, Space, Table, Typography, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { User } from '../api/types';
import { getProject } from '../api/projects';
import { listAllUsers, listUsersByProject, setUserRole } from '../api/users';
import { useAuth } from '../auth/AuthContext';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

export function UsersPage() {
  const { claims } = useAuth();
  const isSuperadmin = claims?.role === 'superadmin';
  const queryClient = useQueryClient();

  // A regular admin's project comes from their own JWT, not a picker —
  // they were never allowed to see another project's users anyway
  // (GET /projects/{id}/users 403s for any project but their own), so a
  // dropdown fed by GET /projects (which lists every project, since that
  // endpoint has no per-admin scoping) only ever showed choices that would
  // fail on selection. Fixed by dropping the dropdown for non-superadmins.
  const ownProjectId = claims?.project_id ?? null;

  const { data: ownProject } = useQuery({
    queryKey: ['project', ownProjectId],
    queryFn: () => getProject(ownProjectId!),
    enabled: !isSuperadmin && !!ownProjectId,
  });

  const {
    data: scopedUsers,
    isLoading: scopedUsersLoading,
    isError: scopedIsError,
    error: scopedError,
  } = useQuery({
    queryKey: ['project-users', ownProjectId],
    queryFn: () => listUsersByProject(ownProjectId!),
    enabled: !isSuperadmin && !!ownProjectId,
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
        queryClient.invalidateQueries({ queryKey: ['project-users', ownProjectId] });
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
      {ownProject && (
        <Typography.Title level={5} style={{ marginBottom: 12 }}>
          {ownProject.name}
        </Typography.Title>
      )}
      <Table rowKey="id" loading={usersLoading} dataSource={users} columns={columns} />
    </>
  );
}
