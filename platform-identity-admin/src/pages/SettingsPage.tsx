import { useEffect, useState } from 'react';
import { Button, Card, Form, InputNumber, message, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { getJwtTtlMinutes, setJwtTtlMinutes } from '../api/settings';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

const { Text } = Typography;

interface TTLFormValues {
  days: number;
}

export function SettingsPage() {
  const queryClient = useQueryClient();
  const [form] = Form.useForm<TTLFormValues>();
  const [currentDays, setCurrentDays] = useState<number | null>(null);

  const { data: ttlMinutes, isLoading, isError, error } = useQuery({
    queryKey: ['jwt-ttl'],
    queryFn: () => getJwtTtlMinutes(),
  });
  useQueryErrorToast(isError, error);

  useEffect(() => {
    if (ttlMinutes != null) {
      const days = Math.round((ttlMinutes / (60 * 24)) * 10) / 10;
      form.setFieldsValue({ days });
      setCurrentDays(days);
    }
  }, [ttlMinutes, form]);

  const mutation = useMutation({
    mutationFn: (values: TTLFormValues) => setJwtTtlMinutes(Math.round(values.days * 60 * 24)),
    onSuccess: () => {
      message.success('Token vaxtı yeniləndi — dərhal tətbiq olunur, restart lazım deyil');
      queryClient.invalidateQueries({ queryKey: ['jwt-ttl'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  return (
    <Card title="Giriş tokeninin müddəti" loading={isLoading} style={{ maxWidth: 480 }}>
      <Text type="secondary">
        İstifadəçilər login olanda aldıqları token neçə gün etibarlı olsun. Dəyişiklik yalnız bundan sonrakı
        yeni login-lərə tətbiq olunur — hazırda daxil olmuş istifadəçilərin token-i öz köhnə müddətini saxlayır.
      </Text>
      <Form
        form={form}
        layout="vertical"
        style={{ marginTop: 16 }}
        onFinish={(values) => mutation.mutate(values)}
      >
        <Form.Item
          name="days"
          label="Gün sayı"
          rules={[{ required: true, type: 'number', min: 0.01, message: 'Müsbət bir dəyər olmalıdır' }]}
        >
          <InputNumber min={0.01} step={1} style={{ width: '100%' }} />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={mutation.isPending} disabled={currentDays === null}>
          Yadda saxla
        </Button>
      </Form>
    </Card>
  );
}
