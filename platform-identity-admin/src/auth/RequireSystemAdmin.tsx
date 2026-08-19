import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

// A second, narrower gate for pages that are system-admin-only (Shop-lar,
// Product-lar, İstifadəçilər) — a shop-scoped user already passed
// RequireAuth via their shop membership, but must not reach these pages,
// since the backend would reject every call from them with 403 anyway.
export function RequireSystemAdmin() {
  const { claims, myShopId } = useAuth();
  const isSystemAdmin = claims?.system_role === 'admin' || claims?.system_role === 'superadmin';
  if (!isSystemAdmin) {
    return <Navigate to={myShopId ? `/shops/${myShopId}/members` : '/login'} replace />;
  }
  return <Outlet />;
}
