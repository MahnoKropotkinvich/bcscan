import React, { useEffect, useState, useCallback } from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { Layout, Menu, Badge, Tooltip, Dropdown, Button, Spin, message } from 'antd';
import {
  DashboardOutlined,
  AlertOutlined,
  BellOutlined,
  FileTextOutlined,
  BarChartOutlined,
  HeartOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  TeamOutlined,
  AuditOutlined,
  BlockOutlined,
} from '@ant-design/icons';
import Dashboard from './pages/Dashboard';
import RuleManagement from './pages/RuleManagement';
import SystemStatus from './pages/SystemStatus';
import LoginPage from './pages/LoginPage';
import AlertCenter from './pages/AlertCenter';
import ReportCenter from './pages/ReportCenter';
import UserManagement from './pages/UserManagement';
import AuditLog from './pages/AuditLog';
import TransactionExplorer from './pages/TransactionExplorer';
import { getHealth, getCurrentUser, getToken, removeToken, getAlertStats, wsClient, AuthUser, ROLE_LABELS } from './api';
import './App.css';

const { Header, Content, Sider } = Layout;

// ==================== RBAC 菜单配置 ====================

interface MenuItem {
  key: string;
  path: string;
  icon: React.ReactNode;
  label: string;
  roles?: string[]; // 为空表示所有角色可见
}

const ALL_MENU_ITEMS: MenuItem[] = [
  { key: 'dashboard', path: '/', icon: <DashboardOutlined />, label: '仪表板' },
  { key: 'alerts', path: '/alerts', icon: <AlertOutlined />, label: '告警中心' },
  { key: 'explorer', path: '/explorer', icon: <BlockOutlined />, label: '交易浏览器' },
  { key: 'reports', path: '/reports', icon: <BarChartOutlined />, label: '报告中心' },
  { key: 'rules', path: '/rules', icon: <FileTextOutlined />, label: '规则管理', roles: ['admin', 'analyst', 'developer'] },
  { key: 'users', path: '/users', icon: <TeamOutlined />, label: '用户管理', roles: ['admin'] },
  { key: 'audit', path: '/audit', icon: <AuditOutlined />, label: '审计日志', roles: ['admin', 'operator'] },
  { key: 'system', path: '/system', icon: <SettingOutlined />, label: '系统状态', roles: ['admin', 'operator'] },
];

// 根据角色过滤菜单
function getMenuForRole(role: string): MenuItem[] {
  return ALL_MENU_ITEMS.filter((item) => {
    if (!item.roles) return true; // 无角色限制 => 所有人可见
    if (role === 'admin') return true; // admin 看到一切
    return item.roles.includes(role);
  });
}

// ==================== 导航菜单 ====================

interface NavigationMenuProps {
  role: string;
  pendingAlerts: number;
}

const NavigationMenu: React.FC<NavigationMenuProps> = ({ role, pendingAlerts }) => {
  const location = useLocation();
  const menuItems = getMenuForRole(role);

  const pathToKey: Record<string, string> = {};
  menuItems.forEach((item) => {
    pathToKey[item.path] = item.key;
  });

  const selectedKey = pathToKey[location.pathname] || 'dashboard';

  return (
    <Menu
      mode="inline"
      selectedKeys={[selectedKey]}
      style={{ height: '100%', borderRight: 0 }}
      items={menuItems.map((item) => ({
        key: item.key,
        icon: item.icon,
        label: (
          <Link to={item.path}>
            {item.label}
            {item.key === 'alerts' && pendingAlerts > 0 && (
              <Badge count={pendingAlerts} size="small" style={{ marginLeft: 8 }} />
            )}
          </Link>
        ),
      }))}
    />
  );
};

// ==================== 主布局（已登录） ====================

interface MainLayoutProps {
  user: AuthUser;
  onLogout: () => void;
  systemOk: boolean | null;
  pendingAlerts: number;
}

const MainLayout: React.FC<MainLayoutProps> = ({ user, onLogout, systemOk, pendingAlerts }) => {
  const roleLabel = ROLE_LABELS[user.role] || user.role;

  const userMenuItems = [
    {
      key: 'info',
      label: (
        <span style={{ color: '#999' }}>
          {roleLabel} · {user.email}
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
          {/* 未读告警 Badge */}
          {pendingAlerts > 0 && (
            <Link to="/alerts">
              <Badge count={pendingAlerts} overflowCount={99}>
                <BellOutlined style={{ color: '#fff', fontSize: 18 }} />
              </Badge>
            </Link>
          )}

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
          <NavigationMenu role={user.role} pendingAlerts={pendingAlerts} />
        </Sider>
        <Layout style={{ padding: '24px' }}>
          <Content style={{ background: '#fff', padding: 24, margin: 0, minHeight: 280, borderRadius: 6 }}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/alerts" element={<AlertCenter />} />
              <Route path="/explorer" element={<TransactionExplorer />} />
              <Route path="/reports" element={<ReportCenter />} />
              <Route path="/rules" element={<RuleManagement />} />
              <Route path="/users" element={<UserManagement />} />
              <Route path="/audit" element={<AuditLog />} />
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
  const [pendingAlerts, setPendingAlerts] = useState(0);

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

  // 轮询未读告警数
  useEffect(() => {
    if (!user) return;

    const fetchPendingAlerts = async () => {
      try {
        const res = await getAlertStats();
        if (res.data.success) {
          setPendingAlerts(res.data.data.pending_count || 0);
        }
      } catch {
        // 忽略
      }
    };

    fetchPendingAlerts();
    const interval = setInterval(fetchPendingAlerts, 15000);
    return () => clearInterval(interval);
  }, [user]);

  // 浏览器通知 + WebSocket 实时告警
  useEffect(() => {
    if (!user) return;

    // 请求浏览器通知权限
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }

    wsClient.connect();
    const unsubscribe = wsClient.onMessage((event) => {
      // 更新未读告警数
      setPendingAlerts((prev) => prev + 1);

      // 浏览器通知
      if ('Notification' in window && Notification.permission === 'granted') {
        const severityLabel: Record<string, string> = {
          critical: '🔴 严重', high: '🟠 高危', medium: '🔵 中危', low: '🟢 低危',
        };
        const title = `${severityLabel[event.severity] || event.severity} 风险告警`;
        const body = `${event.event_type}\n${event.description || ''}`.slice(0, 100);
        try {
          const notification = new Notification(title, {
            body,
            icon: '/favicon.ico',
            tag: `risk-${event.id}`, // 同 tag 去重
          });
          notification.onclick = () => {
            window.focus();
            notification.close();
          };
          // 5秒后自动关闭
          setTimeout(() => notification.close(), 5000);
        } catch { /* iOS Safari 不支持 new Notification */ }
      }
    });

    return () => {
      unsubscribe();
    };
  }, [user]);

  const handleLoginSuccess = () => {
    checkAuth();
  };

  const handleLogout = () => {
    removeToken();
    setUser(null);
    message.success('已退出登录');
  };

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
        <MainLayout user={user} onLogout={handleLogout} systemOk={systemOk} pendingAlerts={pendingAlerts} />
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
