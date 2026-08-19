import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

export function RequireAuth() {
  const { token, claims } = useAuth();
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  if (claims?.system_role !== 'admin' && claims?.system_role !== 'superadmin') {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
