export interface ApiError {
  code: string;
  message: string;
}

export interface Shop {
  id: string;
  name: string;
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
  created_at: string;
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
}

export interface JwtClaims {
  user_id: string;
  system_role: string;
  exp: number;
  iat?: number;
}
