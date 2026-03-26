import React, { useEffect, useState, useCallback } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { Layout, Menu, Badge, Tooltip, Dropdown, Button, Spin, message } from 'antd';
import {
  DashboardOutlined,
  AlertOutlined,
  FileTextOutlined,
  HeartOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import Dashboard from './pages/Dashboard';
import RiskEvents from './pages/RiskEvents';
import RuleManagement from './pages/RuleManagement';
import SystemStatus from './pages/SystemStatus';
import LoginPage from './pages/LoginPage';
import { getHealth, getCurrentUser, getToken, removeToken, AuthUser } from './api';
import './App.css';

const { Header, Content, Sider } = Layout;

// ==================== 导航菜单 ====================

const NavigationMenu: React.FC = () => {
  const location = useLocation();

  const pathToKey: Record<string, string> = {
    '/': 'dashboard',
    '/risks': 'risks',
    '/rules': 'rules',
    '/system': 'system',
  };

  const selectedKey = pathToKey[location.pathname] || 'dashboard';

  return (
    <Menu
      mode="inline"
      selectedKeys={[selectedKey]}
      style={{ height: '100%', borderRight: 0 }}
      items={[
        {
          key: 'dashboard',
          icon: <DashboardOutlined />,
          label: <Link to="/">仪表板</Link>,
        },
        {
          key: 'risks',
          icon: <AlertOutlined />,
          label: <Link to="/risks">风险事件</Link>,
        },
        {
          key: 'rules',
          icon: <FileTextOutlined />,
          label: <Link to="/rules">规则管理</Link>,
        },
        {
          key: 'system',
          icon: <SettingOutlined />,
          label: <Link to="/system">系统状态</Link>,
        },
      ]}
    />
  );
};

// ==================== 主布局（已登录） ====================

interface MainLayoutProps {
  user: AuthUser;
  onLogout: () => void;
  systemOk: boolean | null;
}

const MainLayout: React.FC<MainLayoutProps> = ({ user, onLogout, systemOk }) => {
  const userMenuItems = [
    {
      key: 'info',
      label: (
        <span style={{ color: '#999' }}>
          {user.role === 'admin' ? '管理员' : '用户'} · {user.email}
        </span>
      ),
      disabled: true,
    },
    { type: 'divider' as const },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: onLogout,
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
        <div style={{ color: 'white', fontSize: '18px', fontWeight: 'bold' }}>
          <HeartOutlined style={{ marginRight: 8 }} /> SCRRMS - 智能合约运行时风险监控系统
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <Tooltip title={systemOk === null ? '检查中...' : systemOk ? '系统正常' : '系统异常'}>
            <Badge
              status={systemOk === null ? 'processing' : systemOk ? 'success' : 'error'}
              text={
                <span style={{ color: '#fff', fontSize: 13 }}>
                  {systemOk === null ? '检查中' : systemOk ? '系统正常' : '系统异常'}
                </span>
              }
            />
          </Tooltip>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <Button type="text" icon={<UserOutlined />} style={{ color: '#fff' }}>
              {user.username}
            </Button>
          </Dropdown>
        </div>
      </Header>
      <Layout>
        <Sider width={200} theme="light">
          <NavigationMenu />
        </Sider>
        <Layout style={{ padding: '24px' }}>
          <Content style={{ background: '#fff', padding: 24, margin: 0, minHeight: 280, borderRadius: 6 }}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/risks" element={<RiskEvents />} />
              <Route path="/rules" element={<RuleManagement />} />
              <Route path="/system" element={<SystemStatus />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

// ==================== 应用入口 ====================

const App: React.FC = () => {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [authChecked, setAuthChecked] = useState(false);
  const [systemOk, setSystemOk] = useState<boolean | null>(null);

  // 检查是否已登录
  const checkAuth = useCallback(async () => {
    const token = getToken();
    if (!token) {
      setAuthChecked(true);
      return;
    }

    try {
      const res = await getCurrentUser();
      if (res.data.success) {
        setUser(res.data.data);
      }
    } catch {
      removeToken();
    } finally {
      setAuthChecked(true);
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  // 系统健康检查
  useEffect(() => {
    const check = async () => {
      try {
        const res = await getHealth();
        setSystemOk(res.data.status === 'ok');
      } catch {
        setSystemOk(false);
      }
    };
    check();
    const interval = setInterval(check, 30000);
    return () => clearInterval(interval);
  }, []);

  const handleLoginSuccess = () => {
    checkAuth();
  };

  const handleLogout = () => {
    removeToken();
    setUser(null);
    message.success('已退出登录');
  };

  // 还在检查登录状态 —— 显示全屏 loading 而不是空白
  if (!authChecked) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <Router>
      {user ? (
        <MainLayout user={user} onLogout={handleLogout} systemOk={systemOk} />
      ) : (
        <Routes>
          <Route path="/login" element={<LoginPage onLoginSuccess={handleLoginSuccess} />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      )}
    </Router>
  );
};

export default App;
