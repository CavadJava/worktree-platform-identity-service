import { platformIdentityApi } from './httpClients';
import type { Member, User } from './types';

export async function listAllUsers(): Promise<User[]> {
  const response = await platformIdentityApi.get<User[]>('/users');
  return response.data;
}

export async function getUser(id: string): Promise<User> {
  const response = await platformIdentityApi.get<User>(`/users/${id}`);
  return response.data;
}

export async function createUser(
  name: string,
  username: string,
  email: string,
  password: string,
  systemRole: string
): Promise<User> {
  const response = await platformIdentityApi.post<User>('/users', {
    name,
    username,
    email,
    password,
    system_role: systemRole,
  });
  return response.data;
}

export async function setSystemRole(userId: string, systemRole: string): Promise<User> {
  const response = await platformIdentityApi.post<User>(`/users/${userId}/system-role`, { system_role: systemRole });
  return response.data;
}

export async function setStatus(userId: string, status: string): Promise<User> {
  const response = await platformIdentityApi.post<User>(`/users/${userId}/status`, { status });
  return response.data;
}

export interface ProfileUpdateInput {
  name?: string;
  email?: string;
  password?: string;
}

export async function updateProfile(userId: string, update: ProfileUpdateInput): Promise<User> {
  const response = await platformIdentityApi.post<User>(`/users/${userId}/profile`, update);
  return response.data;
}

export async function listUserShops(userId: string): Promise<Member[]> {
  const response = await platformIdentityApi.get<Member[]>(`/users/${userId}/shops`);
  return response.data;
}

export async function listShopMembers(shopId: string): Promise<Member[]> {
  const response = await platformIdentityApi.get<Member[]>(`/shops/${shopId}/members`);
  return response.data;
}

export async function addShopMember(shopId: string, userId: string, shopRole: string): Promise<Member> {
  const response = await platformIdentityApi.post<Member>(`/shops/${shopId}/members`, {
    user_id: userId,
    shop_role: shopRole,
  });
  return response.data;
}

export async function setMemberRole(shopId: string, userId: string, shopRole: string): Promise<Member> {
  const response = await platformIdentityApi.post<Member>(`/shops/${shopId}/members/${userId}/role`, {
    shop_role: shopRole,
  });
  return response.data;
}

export async function removeShopMember(shopId: string, userId: string): Promise<void> {
  await platformIdentityApi.delete(`/shops/${shopId}/members/${userId}`);
}

export async function updateMemberProfile(shopId: string, userId: string, update: ProfileUpdateInput): Promise<User> {
  const response = await platformIdentityApi.post<User>(`/shops/${shopId}/members/${userId}/profile`, update);
  return response.data;
}

export async function addNewShopMember(
  shopId: string,
  name: string,
  username: string,
  email: string,
  password: string
): Promise<Member> {
  const response = await platformIdentityApi.post<Member>(`/shops/${shopId}/members/new`, {
    name,
    username,
    email,
    password,
  });
  return response.data;
}
