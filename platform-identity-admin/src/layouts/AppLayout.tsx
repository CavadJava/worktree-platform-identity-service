import { useState } from 'react';
import { Drawer, Grid, Layout, Menu, Space } from 'antd';
import type { MenuProps } from 'antd';
import { AppstoreOutlined, MenuOutlined, SettingOutlined, ShoppingOutlined, TeamOutlined, LogoutOutlined, UserOutlined } from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

const { Sider, Header, Content } = Layout;
const { useBreakpoint } = Grid;

export function AppLayout() {
  const { logout, claims, myShopId } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const screens = useBreakpoint();
  const isMobile = !screens.md;
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

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
        { key: '/settings', icon: <SettingOutlined />, label: 'Ayarlar' },
      ]
    : isAdminOrAbove
      ? [{ key: '/products', icon: <ShoppingOutlined />, label: 'Product-lar' }]
      : [{ key: `/shops/${myShopId}/members`, icon: <TeamOutlined />, label: 'Mənim mağazam' }];

  // Highlight the top-level nav item even on a nested route like
  // /shops/:id/members.
  const selectedKey = items?.find((item) => item && location.pathname.startsWith(String(item.key)))?.key as string | undefined;

  const menu = (
    <Menu
      theme="light"
      mode="inline"
      selectedKeys={selectedKey ? [selectedKey] : []}
      items={items}
      onClick={({ key }) => {
        navigate(key);
        setMobileMenuOpen(false);
      }}
    />
  );

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {!isMobile && (
        <Sider theme="light" width={220}>
          <div style={{ padding: 16, fontWeight: 600, fontSize: 16 }}>Teslahubs Admin</div>
          {menu}
        </Sider>
      )}
      {isMobile && (
        <Drawer
          placement="left"
          open={mobileMenuOpen}
          onClose={() => setMobileMenuOpen(false)}
          title="Teslahubs Admin"
          styles={{ body: { padding: 0 } }}
          width={220}
        >
          {menu}
        </Drawer>
      )}
      <Layout>
        <Header
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: 16,
            background: '#fff',
            paddingInline: isMobile ? 12 : 24,
          }}
        >
          {isMobile ? (
            <MenuOutlined style={{ fontSize: 18 }} onClick={() => setMobileMenuOpen(true)} />
          ) : (
            <span />
          )}
          <Space size={16}>
            <span>
              <UserOutlined /> Admin
            </span>
            <a onClick={logout}>
              <LogoutOutlined /> Çıxış
            </a>
          </Space>
        </Header>
        <Content style={{ margin: isMobile ? 12 : 24 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
