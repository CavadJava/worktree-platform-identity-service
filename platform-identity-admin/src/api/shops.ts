import { platformIdentityApi } from './httpClients';
import type { Shop } from './types';

export async function listShops(): Promise<Shop[]> {
  const response = await platformIdentityApi.get<Shop[]>('/shops');
  return response.data;
}

export async function createShop(name: string, shopType: string): Promise<Shop> {
  const response = await platformIdentityApi.post<Shop>('/shops', { name, shop_type: shopType });
  return response.data;
}

export async function getShop(id: string): Promise<Shop> {
  const response = await platformIdentityApi.get<Shop>(`/shops/${id}`);
  return response.data;
}

export async function updateShopProfile(
  id: string,
  contactEmail: string,
  contactPhone: string,
  address: string,
  workHours: string
): Promise<Shop> {
  const response = await platformIdentityApi.post<Shop>(`/shops/${id}/profile`, {
    contact_email: contactEmail,
    contact_phone: contactPhone,
    address,
    work_hours: workHours,
  });
  return response.data;
}
