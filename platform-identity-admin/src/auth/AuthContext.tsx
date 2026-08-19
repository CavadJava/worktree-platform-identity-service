import { createContext, useCallback, useContext, useState, type ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { decodeToken, isExpired } from './jwt';
import type { JwtClaims } from '../api/types';

interface AuthState {
  token: string | null;
  claims: JwtClaims | null;
  // A plain 'user' with no system_role of admin/superadmin can still be
  // let into the panel if they belong to at least one shop — set once
  // after login by a GET /users/{id}/shops lookup (there's no shop
  // membership in the JWT itself, since a user can belong to several
  // shops with different roles, so a single claim wouldn't fit).
  myShopId: string | null;
  myShopRole: string | null;
}

interface AuthContextValue extends AuthState {
  login: (token: string) => void;
  logout: () => void;
  setMyShop: (shopId: string | null, shopRole: string | null) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function readStoredToken(): Pick<AuthState, 'token' | 'claims'> {
  const token = localStorage.getItem('token');
  if (!token) {
    return { token: null, claims: null };
  }
  try {
    const claims = decodeToken(token);
    if (isExpired(claims)) {
      localStorage.removeItem('token');
      return { token: null, claims: null };
    }
    return { token, claims };
  } catch {
    localStorage.removeItem('token');
    return { token: null, claims: null };
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(() => ({ ...readStoredToken(), myShopId: null, myShopRole: null }));
  const queryClient = useQueryClient();

  const login = useCallback((token: string) => {
    localStorage.setItem('token', token);
    setState({ token, claims: decodeToken(token), myShopId: null, myShopRole: null });
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    queryClient.clear();
    setState({ token: null, claims: null, myShopId: null, myShopRole: null });
  }, [queryClient]);

  const setMyShop = useCallback((shopId: string | null, shopRole: string | null) => {
    setState((prev) => ({ ...prev, myShopId: shopId, myShopRole: shopRole }));
  }, []);

  return (
    <AuthContext.Provider
      value={{ token: state.token, claims: state.claims, myShopId: state.myShopId, myShopRole: state.myShopRole, login, logout, setMyShop }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
