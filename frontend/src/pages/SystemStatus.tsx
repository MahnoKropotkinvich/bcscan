import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Tag, Descriptions, Spin, Button, message } from 'antd';
import { ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, ApiOutlined, DatabaseOutlined, CloudServerOutlined } from '@ant-design/icons';
import { getHealth, HealthStatus } from '../api';

const SystemStatus: React.FC = () => {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchHealth = async () => {
    setLoading(true);
    try {
      const res = await getHealth();
      setHealth(res.data);
    } catch (err) {
      console.error('Failed to fetch health:', err);
      setHealth({
        status: 'error',
        timestamp: new Date().toISOString(),
        services: { api: 'unreachable' },
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 15000);
    return () => clearInterval(interval);
  }, []);

  const getStatusIcon = (status: string) => {
    if (status === 'ok') {
      return <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 24 }} />;
    }
    return <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 24 }} />;
  };

  const serviceIcons: Record<string, React.ReactNode> = {
    database: <DatabaseOutlined style={{ fontSize: 20, marginRight: 8 }} />,
    redis: <CloudServerOutlined style={{ fontSize: 20, marginRight: 8 }} />,
    api: <ApiOutlined style={{ fontSize: 20, marginRight: 8 }} />,
  };

  if (loading && !health) {
    return (
      <div style={{ textAlign: 'center', padding: '100px' }}>
        <Spin size="large" tip="检查系统状态..." />
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>系统状态</h2>
        <Button icon={<ReloadOutlined />} onClick={fetchHealth} loading={loading}>
          刷新
        </Button>
      </div>

      <Card style={{ marginBottom: 24 }}>
        <Descriptions bordered column={2}>
          <Descriptions.Item label="系统状态">
            <Tag
              icon={health?.status === 'ok' ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
              color={health?.status === 'ok' ? 'success' : 'error'}
              style={{ fontSize: 14 }}
            >
              {health?.status === 'ok' ? '正常运行' : '服务异常'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="检查时间">
            {health?.timestamp ? new Date(health.timestamp).toLocaleString('zh-CN') : '-'}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <h3 style={{ marginBottom: 16 }}>服务健康状态</h3>
      <Row gutter={16}>
        {health?.services &&
          Object.entries(health.services).map(([name, status]) => (
            <Col span={8} key={name}>
              <Card>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    {serviceIcons[name] || <ApiOutlined style={{ fontSize: 20, marginRight: 8 }} />}
                    <div>
                      <div style={{ fontWeight: 'bold', fontSize: 16, textTransform: 'capitalize' }}>{name}</div>
                      <div style={{ color: '#999', fontSize: 12 }}>
                        {status === 'ok' ? '运行正常' : status}
                      </div>
                    </div>
                  </div>
                  {getStatusIcon(status)}
                </div>
              </Card>
            </Col>
          ))}
      </Row>

      <Card title="系统信息" style={{ marginTop: 24 }}>
        <Descriptions bordered size="small" column={2}>
          <Descriptions.Item label="API 地址">http://localhost:8080</Descriptions.Item>
          <Descriptions.Item label="WebSocket">ws://localhost:8080/api/ws</Descriptions.Item>
          <Descriptions.Item label="数据库">PostgreSQL 14</Descriptions.Item>
          <Descriptions.Item label="缓存">Redis 7</Descriptions.Item>
          <Descriptions.Item label="消息队列">Redpanda (Kafka)</Descriptions.Item>
          <Descriptions.Item label="区块链节点">Ganache (测试网)</Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );
};

export default SystemStatus;
