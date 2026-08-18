import { platformIdentityApi } from './httpClients';

export async function login(identifier: string, password: string): Promise<{ token: string }> {
  const response = await platformIdentityApi.post<{ token: string }>('/auth/login', { identifier, password });
  return response.data;
}
