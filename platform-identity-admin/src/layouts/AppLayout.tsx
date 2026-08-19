import { Layout, Menu } from 'antd';
import type { MenuProps } from 'antd';
import { AppstoreOutlined, ShoppingOutlined, TeamOutlined, LogoutOutlined, UserOutlined } from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

const { Sider, Header, Content } = Layout;

const items: MenuProps['items'] = [
  { key: '/shops', icon: <AppstoreOutlined />, label: 'Shop-lar' },
  { key: '/products', icon: <ShoppingOutlined />, label: 'Product-lar' },
  { key: '/users', icon: <TeamOutlined />, label: 'İstifadəçilər' },
];

export function AppLayout() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

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
