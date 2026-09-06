import { platformIdentityApi } from './httpClients';
import type { Product, Subproject } from './types';

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

export async function updateProductProfile(
  productId: string,
  description: string,
  techStack: string
): Promise<Product> {
  const response = await platformIdentityApi.post<Product>(`/products/${productId}/profile`, {
    description,
    tech_stack: techStack,
  });
  return response.data;
}

export async function listSubprojects(productId: string): Promise<Subproject[]> {
  const response = await platformIdentityApi.get<Subproject[]>(`/products/${productId}/subprojects`);
  return response.data;
}

export async function addSubproject(productId: string, name: string, description: string): Promise<Subproject> {
  const response = await platformIdentityApi.post<Subproject>(`/products/${productId}/subprojects`, {
    name,
    description,
  });
  return response.data;
}

export async function removeSubproject(productId: string, subId: string): Promise<void> {
  await platformIdentityApi.delete(`/products/${productId}/subprojects/${subId}`);
}
