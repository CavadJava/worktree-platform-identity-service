import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

// Products-lar is open to admin and superadmin alike — an admin's view is
// scoped server-side (GET /products/mine) to only the products they hold
// a subscription to, so no further client-side restriction is needed here
// beyond "can this role reach the page at all."
export function RequireAdminOrAbove() {
  const { claims, myShopId } = useAuth();
  const isAdminOrAbove = claims?.system_role === 'admin' || claims?.system_role === 'superadmin';
  if (!isAdminOrAbove) {
    return <Navigate to={myShopId ? `/shops/${myShopId}/members` : '/login'} replace />;
  }
  return <Outlet />;
}
