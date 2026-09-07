import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

// Shops-lar and İstifadəçilər are now superadmin-only — an admin, even one
// who manages several products, has no reason to see every shop or every
// user's shop memberships. Replaces the old RequireSystemAdmin for these
// two pages specifically.
export function RequireSuperadmin() {
  const { claims, myShopId } = useAuth();
  const isSuperadmin = claims?.system_role === 'superadmin';
  if (!isSuperadmin) {
    return <Navigate to={myShopId ? `/shops/${myShopId}/members` : '/products'} replace />;
  }
  return <Outlet />;
}
