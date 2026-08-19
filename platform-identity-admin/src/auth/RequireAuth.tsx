import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

export function RequireAuth() {
  const { token, claims, myShopId } = useAuth();
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  const isSystemAdmin = claims?.system_role === 'admin' || claims?.system_role === 'superadmin';
  // A plain 'user' earns entry only via a shop membership resolved at
  // login time (LoginPage sets myShopId) — a page refresh with a stored
  // token but no myShopId (not re-resolved yet) falls through to login,
  // since there's no cheap way to re-derive it without another request.
  if (!isSystemAdmin && !myShopId) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
