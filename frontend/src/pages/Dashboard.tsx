import React, { useEffect, useState, useCallback, useRef } from 'react';
import { Card, Row, Col, Statistic, Table, Radio, Tag, Spin } from 'antd';
import { AlertOutlined, RiseOutlined, FieldTimeOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { getStats, getTrend, getRiskEvents, wsClient, RiskStats, TrendPoint, RiskEvent } from '../api';

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<RiskStats | null>(null);
  const [trend, setTrend] = useState<TrendPoint[]>([]);
  const [recentEvents, setRecentEvents] = useState<RiskEvent[]>([]);
  const [timeRange, setTimeRange] = useState<string>('10m');
  const [loading, setLoading] = useState(true);
  const wsConnected = useRef(false);

  // 获取统计数据
  const fetchStats = useCallback(async () => {
    try {
      const res = await getStats();
      if (res.data.success) {
        setStats(res.data.data);
      }
    } catch (err) {
      console.error('Failed to fetch stats:', err);
    }
  }, []);

  // 获取趋势数据
  const fetchTrend = useCallback(async () => {
    try {
      const res = await getTrend(timeRange);
      if (res.data.success) {
        setTrend(res.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch trend:', err);
    }
  }, [timeRange]);

  // 获取最近事件
  const fetchRecentEvents = useCallback(async () => {
    try {
      const res = await getRiskEvents({ page_size: 5 });
      setRecentEvents(res.data.items || []);
    } catch (err) {
      console.error('Failed to fetch recent events:', err);
    }
  }, []);

  // 初始加载
  useEffect(() => {
    const loadAll = async () => {
      setLoading(true);
      await Promise.all([fetchStats(), fetchTrend(), fetchRecentEvents()]);
      setLoading(false);
    };
    loadAll();
  }, [fetchStats, fetchTrend, fetchRecentEvents]);

  // 定时刷新（15秒）
  useEffect(() => {
    const interval = setInterval(() => {
      fetchStats();
      fetchTrend();
      fetchRecentEvents();
    }, 15000);
    return () => clearInterval(interval);
  }, [fetchStats, fetchTrend, fetchRecentEvents]);

  // 切换时间范围时重新获取趋势
  useEffect(() => {
    fetchTrend();
  }, [timeRange, fetchTrend]);

  // WebSocket 静默更新（无 popup）
  useEffect(() => {
    if (wsConnected.current) return;
    wsConnected.current = true;

    wsClient.connect();
    const unsubscribe = wsClient.onMessage(() => {
      // 静默刷新数据，不弹 popup
      fetchStats();
      fetchRecentEvents();
    });
    return () => {
      unsubscribe();
      wsConnected.current = false;
    };
  }, []); // 空依赖，只连接一次

  const severityColors: Record<string, string> = {
    critical: '#cf1322',
    high: '#fa8c16',
    medium: '#1890ff',
    low: '#52c41a',
  };

  const columns = [
    { title: '事件类型', dataIndex: 'event_type', key: 'event_type' },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => (
        <Tag color={severityColors[severity] || 'default'}>{severity.toUpperCase()}</Tag>
      ),
    },
    {
      title: '风险分数',
      dataIndex: 'score',
      key: 'score',
      render: (score: number) => {
        const color = score >= 80 ? '#cf1322' : score >= 60 ? '#fa8c16' : '#1890ff';
        return <span style={{ color, fontWeight: 'bold' }}>{score}</span>;
      },
    },
    {
      title: '交易哈希',
      dataIndex: 'tx_hash',
      key: 'tx_hash',
      ellipsis: true,
      render: (hash: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: '12px' }}>{hash}</span>
      ),
    },
    {
      title: '检测时间',
      dataIndex: 'detected_at',
      key: 'detected_at',
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
  ];

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>风险监控仪表板</h2>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总风险事件"
              value={stats?.total || 0}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="严重/高危"
              value={(stats?.by_severity?.critical || 0) + (stats?.by_severity?.high || 0)}
              valueStyle={{ color: '#cf1322' }}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="最近24小时"
              value={stats?.last_24h || 0}
              valueStyle={{ color: '#fa8c16' }}
              prefix={<FieldTimeOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="最近1小时"
              value={stats?.last_1h || 0}
              valueStyle={{ color: '#1890ff' }}
              prefix={<RiseOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title="风险趋势"
        style={{ marginBottom: 24 }}
        extra={
          <Radio.Group value={timeRange} onChange={(e) => setTimeRange(e.target.value)}>
            <Radio.Button value="10m">10分钟</Radio.Button>
            <Radio.Button value="1h">1小时</Radio.Button>
            <Radio.Button value="6h">6小时</Radio.Button>
            <Radio.Button value="24h">24小时</Radio.Button>
            <Radio.Button value="7d">7天</Radio.Button>
          </Radio.Group>
        }
      >
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={trend}>
            <defs>
              <linearGradient id="colorCritical" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#cf1322" stopOpacity={0.8} />
                <stop offset="95%" stopColor="#cf1322" stopOpacity={0.1} />
              </linearGradient>
              <linearGradient id="colorHigh" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#fa8c16" stopOpacity={0.8} />
                <stop offset="95%" stopColor="#fa8c16" stopOpacity={0.1} />
              </linearGradient>
              <linearGradient id="colorMedium" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#1890ff" stopOpacity={0.8} />
                <stop offset="95%" stopColor="#1890ff" stopOpacity={0.1} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Area type="monotone" dataKey="critical" stroke="#cf1322" fillOpacity={1} fill="url(#colorCritical)" name="严重" stackId="1" />
            <Area type="monotone" dataKey="high" stroke="#fa8c16" fillOpacity={1} fill="url(#colorHigh)" name="高危" stackId="1" />
            <Area type="monotone" dataKey="medium" stroke="#1890ff" fillOpacity={1} fill="url(#colorMedium)" name="中危" stackId="1" />
          </AreaChart>
        </ResponsiveContainer>
      </Card>

      <Card title="最近风险事件">
        <Table
          columns={columns}
          dataSource={recentEvents}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Card>
    </div>
  );
};

export default Dashboard;
