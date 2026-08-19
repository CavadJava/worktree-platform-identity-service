import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider, useAuth } from './auth/AuthContext';
import { RequireAuth } from './auth/RequireAuth';
import { RequireSystemAdmin } from './auth/RequireSystemAdmin';
import { AppLayout } from './layouts/AppLayout';
import { LoginPage } from './pages/LoginPage';
import { ShopsPage } from './pages/ShopsPage';
import { ShopMembersPage } from './pages/ShopMembersPage';
import { ProductsPage } from './pages/ProductsPage';
import { UsersPage } from './pages/UsersPage';

const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false } } });

// A shop-scoped user (no system role) has only one real page — their own
// shop's members — so both the index redirect and the catch-all need to
// send them there instead of the system-admin default of /shops.
function DefaultRedirect() {
  const { claims, myShopId } = useAuth();
  const isSystemAdmin = claims?.system_role === 'admin' || claims?.system_role === 'superadmin';
  if (!isSystemAdmin && myShopId) {
    return <Navigate to={`/shops/${myShopId}/members`} replace />;
  }
  return <Navigate to="/shops" replace />;
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<RequireAuth />}>
              <Route element={<AppLayout />}>
                <Route index element={<DefaultRedirect />} />
                <Route path="/shops/:id/members" element={<ShopMembersPage />} />
                <Route element={<RequireSystemAdmin />}>
                  <Route path="/shops" element={<ShopsPage />} />
                  <Route path="/products" element={<ProductsPage />} />
                  <Route path="/users" element={<UsersPage />} />
                </Route>
              </Route>
            </Route>
            <Route path="*" element={<DefaultRedirect />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}
