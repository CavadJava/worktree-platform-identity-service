import { platformIdentityApi } from './httpClients';
import type { Product } from './types';

export async function listProducts(): Promise<Product[]> {
  const response = await platformIdentityApi.get<Product[]>('/products');
  return response.data;
}

export async function createProduct(name: string): Promise<Product> {
  const response = await platformIdentityApi.post<Product>('/products', { name });
  return response.data;
}

export async function checkAccess(productId: string): Promise<{ access: string }> {
  const response = await platformIdentityApi.get<{ access: string }>(`/products/${productId}/access`);
  return response.data;
}

export async function setSubscription(
  userId: string,
  productId: string,
  subscripted: boolean,
  renewed: boolean
): Promise<{ user_id: string; product_id: string; subscripted: boolean; renewed: boolean }> {
  const response = await platformIdentityApi.post(`/users/${userId}/products/${productId}/subscribe`, {
    subscripted,
    renewed,
  });
  return response.data;
}
