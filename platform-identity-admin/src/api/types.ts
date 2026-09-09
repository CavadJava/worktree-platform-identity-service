export interface ApiError {
  code: string;
  message: string;
}

export interface Shop {
  id: string;
  name: string;
  shop_type: string;
  created_at: string;
}

export interface SystemRole {
  id: number;
  name: string;
}

export interface ShopRole {
  id: number;
  name: string;
}

export interface Product {
  id: string;
  name: string;
  description: string;
  tech_stack: string;
  created_at: string;
}

export interface Subproject {
  id: string;
  product_id: string;
  name: string;
  description: string;
  created_at: string;
}

export interface Subscription {
  user_id: string;
  product_id: string;
  subscripted: boolean;
  renewed: boolean;
  notes: string;
}

export interface ProductAdminRequest {
  id: string;
  product_id: string;
  subject_user_id: string;
  requested_by_user_id: string;
  status: string;
  created_at: string;
}

export interface ProductBrowse {
  id: string;
  name: string;
}

export interface BasicUser {
  id: string;
  name: string;
  username: string;
  email: string;
  system_role: string;
}

export interface ProductCustomer {
  user_id: string;
  name: string;
  username: string;
  email: string;
  system_role: string;
  subscripted: boolean;
  renewed: boolean;
  notes: string;
}

export interface Member {
  id: string;
  user_id: string;
  shop_id: string;
  shop_name: string;
  shop_role: string;
  created_at: string;
}

export interface User {
  id: string;
  name: string;
  username: string;
  email: string;
  system_role: string;
  status: string;
  // Only populated by GET /users (the superadmin system-wide list) —
  // empty array, not undefined, for a user with no shop memberships.
  shops?: Member[];
}

export interface JwtClaims {
  user_id: string;
  system_role: string;
  exp: number;
  iat?: number;
}
