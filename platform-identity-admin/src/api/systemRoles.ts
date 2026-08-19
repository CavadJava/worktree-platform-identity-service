import { platformIdentityApi } from './httpClients';
import type { SystemRole, ShopRole } from './types';

export async function listSystemRoles(): Promise<SystemRole[]> {
  const response = await platformIdentityApi.get<SystemRole[]>('/system-roles');
  return response.data;
}

export async function listShopRoles(): Promise<ShopRole[]> {
  const response = await platformIdentityApi.get<ShopRole[]>('/shop-roles');
  return response.data;
}
