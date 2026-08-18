export interface ApiError {
  code: string;
  message: string;
}

export interface Project {
  id: string;
  name: string;
  created_at: string;
}

export interface Role {
  id: number;
  name: string;
}

export interface User {
  id: string;
  name: string;
  username: string;
  email: string;
  role: string;
}

export interface JwtClaims {
  user_id: string;
  project_id: string;
  role: string;
  exp: number;
  iat?: number;
}
