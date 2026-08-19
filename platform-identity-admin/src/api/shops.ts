import { platformIdentityApi } from './httpClients';
import type { Shop } from './types';

export async function listShops(): Promise<Shop[]> {
  const response = await platformIdentityApi.get<Shop[]>('/shops');
  return response.data;
}

export async function createShop(name: string): Promise<Shop> {
  const response = await platformIdentityApi.post<Shop>('/shops', { name });
  return response.data;
}

export async function getShop(id: string): Promise<Shop> {
  const response = await platformIdentityApi.get<Shop>(`/shops/${id}`);
  return response.data;
}
