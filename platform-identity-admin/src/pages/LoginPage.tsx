import { useState } from 'react';
import { Button, Card, Form, Input, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { login } from '../api/auth';
import { listUserShops } from '../api/users';
import { useAuth } from '../auth/AuthContext';
import { decodeToken } from '../auth/jwt';

interface LoginFormValues {
  identifier: string;
  password: string;
}

export function LoginPage() {
  const [loading, setLoading] = useState(false);
  const { login: setToken, setMyShop } = useAuth();
  const navigate = useNavigate();

  async function onFinish(values: LoginFormValues) {
    setLoading(true);
    try {
      const { token } = await login(values.identifier, values.password);
      const claims = decodeToken(token);

      if (claims.system_role === 'admin' || claims.system_role === 'superadmin') {
        setToken(token);
        navigate('/shops');
        return;
      }

      // Not a system-level admin — check if they're a shop-admin or
      // shop-user of at least one shop, which also earns panel access
      // (scoped to just that shop's members page).
      const memberships = await listUserShops(claims.user_id);
      if (memberships.length === 0) {
        message.error('Yalnız admin, superadmin, ya da bir mağazanın üzvü olan istifadəçilər giriş edə bilər');
        return;
      }

      setToken(token);
      setMyShop(memberships[0].shop_id, memberships[0].shop_role);
      navigate(`/shops/${memberships[0].shop_id}/members`);
    } catch (err) {
      message.error(err instanceof Error ? err.message : 'Login uğursuz oldu');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
      <Card title="Teslahubs Admin — Giriş" style={{ width: 360 }}>
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item name="identifier" label="Username və ya Email" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="Şifrə" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              Giriş
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
