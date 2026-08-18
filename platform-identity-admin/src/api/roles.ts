import { platformIdentityApi } from './httpClients';
import type { Role } from './types';

export async function listRoles(): Promise<Role[]> {
  const response = await platformIdentityApi.get<Role[]>('/roles');
  return response.data;
}
