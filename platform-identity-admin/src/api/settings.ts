import { platformIdentityApi } from './httpClients';

export async function getJwtTtlMinutes(): Promise<number> {
  const response = await platformIdentityApi.get<{ jwt_ttl_minutes: number }>('/settings/jwt-ttl');
  return response.data.jwt_ttl_minutes;
}

export async function setJwtTtlMinutes(minutes: number): Promise<number> {
  const response = await platformIdentityApi.post<{ jwt_ttl_minutes: number }>('/settings/jwt-ttl', {
    jwt_ttl_minutes: minutes,
  });
  return response.data.jwt_ttl_minutes;
}
