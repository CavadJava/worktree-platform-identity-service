import { platformIdentityApi } from './httpClients';
import type { User } from './types';

export async function listUsersByProject(projectId: string): Promise<User[]> {
  const response = await platformIdentityApi.get<User[]>(`/projects/${projectId}/users`);
  return response.data;
}

export async function setUserRole(userId: string, role: string): Promise<User> {
  const response = await platformIdentityApi.post<User>(`/users/${userId}/role`, { role });
  return response.data;
}
