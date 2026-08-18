import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

export function RequireAuth() {
  const { token, claims } = useAuth();
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  if (claims?.role !== 'admin') {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
