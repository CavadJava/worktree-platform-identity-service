import { Table } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { listRoles } from '../api/roles';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

export function RolesPage() {
  const { data: roles, isLoading, isError, error } = useQuery({ queryKey: ['roles'], queryFn: () => listRoles() });
  useQueryErrorToast(isError, error);

  const columns = [
    { title: 'ID', dataIndex: 'id' },
    { title: 'Ad', dataIndex: 'name' },
  ];

  return <Table rowKey="id" loading={isLoading} dataSource={roles} columns={columns} pagination={false} />;
}
