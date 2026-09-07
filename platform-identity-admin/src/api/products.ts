import { platformIdentityApi } from './httpClients';
import type { Product, Subproject, Subscription, ProductAdminRequest, ProductBrowse } from './types';

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
  renewed: boolean,
  notes: string
): Promise<Subscription> {
  const response = await platformIdentityApi.post<Subscription>(`/users/${userId}/products/${productId}/subscribe`, {
    subscripted,
    renewed,
    notes,
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

export async function createProductUser(
  productId: string,
  name: string,
  username: string,
  email: string,
  password: string,
  systemRole: string
): Promise<{ user_id: string; product_id: string; subscripted: boolean; renewed: boolean }> {
  const response = await platformIdentityApi.post(`/products/${productId}/users`, {
    name,
    username,
    email,
    password,
    system_role: systemRole,
  });
  return response.data;
}

export async function listMyProducts(): Promise<Product[]> {
  const response = await platformIdentityApi.get<Product[]>('/products/mine');
  return response.data;
}

export async function listBrowseProducts(): Promise<ProductBrowse[]> {
  const response = await platformIdentityApi.get<ProductBrowse[]>('/products/browse');
  return response.data;
}

export async function requestProductAdmin(productId: string, subjectUserId: string): Promise<ProductAdminRequest> {
  const response = await platformIdentityApi.post<ProductAdminRequest>(`/products/${productId}/admin-requests`, {
    subject_user_id: subjectUserId,
  });
  return response.data;
}

export async function listAdminRequests(productId: string): Promise<ProductAdminRequest[]> {
  const response = await platformIdentityApi.get<ProductAdminRequest[]>(`/products/${productId}/admin-requests`);
  return response.data;
}

export async function decideAdminRequest(productId: string, requestId: string, approve: boolean): Promise<ProductAdminRequest> {
  const response = await platformIdentityApi.post<ProductAdminRequest>(
    `/products/${productId}/admin-requests/${requestId}/decide`,
    { approve }
  );
  return response.data;
}

export async function promoteProductAdmin(productId: string, subjectUserId: string): Promise<Subscription> {
  const response = await platformIdentityApi.post<Subscription>(`/products/${productId}/admin`, {
    subject_user_id: subjectUserId,
  });
  return response.data;
}
