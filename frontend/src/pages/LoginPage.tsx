import React, { useState } from 'react';
import { Card, Form, Input, Button, message, Tabs, Typography, Select } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { login, register, setToken, ROLE_LABELS } from '../api';

const { Title } = Typography;

interface LoginPageProps {
  onLoginSuccess: () => void;
}

const LoginPage: React.FC<LoginPageProps> = ({ onLoginSuccess }) => {
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('login');

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await login(values);
      if (res.data.success) {
        setToken(res.data.data.token);
        message.success(`欢迎回来，${res.data.data.user.username}`);
        onLoginSuccess();
      }
    } catch (err: any) {
      const msg = err.response?.data?.error || '登录失败';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (values: { username: string; email: string; password: string; confirm: string; role?: string }) => {
    if (values.password !== values.confirm) {
      message.error('两次输入的密码不一致');
      return;
    }
    setLoading(true);
    try {
      const res = await register({
        username: values.username,
        email: values.email,
        password: values.password,
        role: values.role,
      });
      if (res.data.success) {
        message.success('注册成功，请登录');
        setActiveTab('login');
      }
    } catch (err: any) {
      const msg = err.response?.data?.error || '注册失败';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        background: 'linear-gradient(135deg, #001529 0%, #003a8c 100%)',
      }}
    >
      <Card style={{ width: 420, borderRadius: 8, boxShadow: '0 4px 24px rgba(0,0,0,0.15)' }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Title level={3} style={{ marginBottom: 4 }}>SCRRMS</Title>
          <div style={{ color: '#999' }}>智能合约运行时风险监控系统</div>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          centered
          items={[
            {
              key: 'login',
              label: '登录',
              children: (
                <Form onFinish={handleLogin} size="large">
                  <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
                    <Input prefix={<UserOutlined />} placeholder="用户名" />
                  </Form.Item>
                  <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
                    <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                  </Form.Item>
                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={loading} block>
                      登录
                    </Button>
                  </Form.Item>
                </Form>
              ),
            },
            {
              key: 'register',
              label: '注册',
              children: (
                <Form onFinish={handleRegister} size="large">
                  <Form.Item
                    name="username"
                    rules={[
                      { required: true, message: '请输入用户名' },
                      { min: 3, message: '用户名至少3个字符' },
                    ]}
                  >
                    <Input prefix={<UserOutlined />} placeholder="用户名" />
                  </Form.Item>
                  <Form.Item
                    name="email"
                    rules={[
                      { required: true, message: '请输入邮箱' },
                      { type: 'email', message: '邮箱格式不正确' },
                    ]}
                  >
                    <Input prefix={<MailOutlined />} placeholder="邮箱" />
                  </Form.Item>
                  <Form.Item
                    name="password"
                    rules={[
                      { required: true, message: '请输入密码' },
                      { min: 6, message: '密码至少6位' },
                    ]}
                  >
                    <Input.Password prefix={<LockOutlined />} placeholder="密码" />
                  </Form.Item>
                  <Form.Item
                    name="confirm"
                    rules={[{ required: true, message: '请确认密码' }]}
                  >
                    <Input.Password prefix={<LockOutlined />} placeholder="确认密码" />
                  </Form.Item>
                  <Form.Item name="role" initialValue="user">
                    <Select
                      prefix={<SafetyCertificateOutlined />}
                      placeholder="选择角色"
                    >
                      {Object.entries(ROLE_LABELS).map(([k, v]) => (
                        <Select.Option key={k} value={k}>{v}</Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={loading} block>
                      注册
                    </Button>
                  </Form.Item>
                </Form>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default LoginPage;
