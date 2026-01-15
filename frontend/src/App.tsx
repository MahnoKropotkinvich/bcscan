import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { Layout, Menu } from 'antd';
import { DashboardOutlined, AlertOutlined, FileTextOutlined } from '@ant-design/icons';
import Dashboard from './pages/Dashboard';
import RiskEvents from './pages/RiskEvents';
import './App.css';

const { Header, Content, Sider } = Layout;

const App: React.FC = () => {
  return (
    <Router>
      <Layout style={{ minHeight: '100vh' }}>
        <Header style={{ display: 'flex', alignItems: 'center' }}>
          <div style={{ color: 'white', fontSize: '20px', fontWeight: 'bold' }}>
            BCScan - 区块链安全监控
          </div>
        </Header>
        <Layout>
          <Sider width={200} theme="light">
            <Menu mode="inline" defaultSelectedKeys={['1']} style={{ height: '100%' }}>
              <Menu.Item key="1" icon={<DashboardOutlined />}>
                <Link to="/">仪表板</Link>
              </Menu.Item>
              <Menu.Item key="2" icon={<AlertOutlined />}>
                <Link to="/risks">风险事件</Link>
              </Menu.Item>
              <Menu.Item key="3" icon={<FileTextOutlined />}>
                <Link to="/rules">规则管理</Link>
              </Menu.Item>
            </Menu>
          </Sider>
          <Layout style={{ padding: '24px' }}>
            <Content style={{ background: '#fff', padding: 24, margin: 0, minHeight: 280 }}>
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/risks" element={<RiskEvents />} />
                <Route path="/rules" element={<div>规则管理（待实现）</div>} />
              </Routes>
            </Content>
          </Layout>
        </Layout>
      </Layout>
    </Router>
  );
};

export default App;
