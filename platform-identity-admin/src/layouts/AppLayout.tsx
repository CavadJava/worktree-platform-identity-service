import { Layout, Menu } from 'antd';
import type { MenuProps } from 'antd';
import { AppstoreOutlined, ShoppingOutlined, TeamOutlined, LogoutOutlined, UserOutlined } from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

const { Sider, Header, Content } = Layout;

export function AppLayout() {
  const { logout, claims, myShopId } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const isSuperadmin = claims?.system_role === 'superadmin';
  const isAdminOrAbove = claims?.system_role === 'admin' || isSuperadmin;

  // A shop-scoped user (no system role) only ever has one page to see —
  // their own shop's members — so the nav is reduced to just that,
  // rather than showing Shop-lar/Product-lar/İstifadəçilər they have no
  // access to anyway. A plain admin sees only Product-lar; Shop-lar and
  // İstifadəçilər are superadmin-only.
  const items: MenuProps['items'] = isSuperadmin
    ? [
        { key: '/shops', icon: <AppstoreOutlined />, label: 'Shop-lar' },
        { key: '/products', icon: <ShoppingOutlined />, label: 'Product-lar' },
        { key: '/users', icon: <TeamOutlined />, label: 'İstifadəçilər' },
      ]
    : isAdminOrAbove
      ? [{ key: '/products', icon: <ShoppingOutlined />, label: 'Product-lar' }]
      : [{ key: `/shops/${myShopId}/members`, icon: <TeamOutlined />, label: 'Mənim mağazam' }];

  // Highlight the top-level nav item even on a nested route like
  // /shops/:id/members.
  const selectedKey = items?.find((item) => item && location.pathname.startsWith(String(item.key)))?.key as string | undefined;

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="light" width={220}>
        <div style={{ padding: 16, fontWeight: 600, fontSize: 16 }}>Teslahubs Admin</div>
        <Menu
          theme="light"
          mode="inline"
          selectedKeys={selectedKey ? [selectedKey] : []}
          items={items}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 16, background: '#fff' }}>
          <span>
            <UserOutlined /> Admin
          </span>
          <a onClick={logout}>
            <LogoutOutlined /> Çıxış
          </a>
        </Header>
        <Content style={{ margin: 24 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
