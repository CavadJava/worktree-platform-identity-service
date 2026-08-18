import { createContext, useCallback, useContext, useState, type ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { decodeToken, isExpired } from './jwt';
import type { JwtClaims } from '../api/types';

interface AuthState {
  token: string | null;
  claims: JwtClaims | null;
}

interface AuthContextValue extends AuthState {
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function readStoredToken(): AuthState {
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
  const [state, setState] = useState<AuthState>(readStoredToken);
  const queryClient = useQueryClient();

  const login = useCallback((token: string) => {
    localStorage.setItem('token', token);
    setState({ token, claims: decodeToken(token) });
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    queryClient.clear();
    setState({ token: null, claims: null });
  }, [queryClient]);

  return (
    <AuthContext.Provider value={{ token: state.token, claims: state.claims, login, logout }}>
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
