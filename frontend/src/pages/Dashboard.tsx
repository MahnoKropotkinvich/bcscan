import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table } from 'antd';
import { AlertOutlined, CheckCircleOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface RiskStats {
  total: number;
  critical: number;
  high: number;
  medium: number;
}

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<RiskStats>({ total: 0, critical: 0, high: 0, medium: 0 });
  const [recentEvents, setRecentEvents] = useState<any[]>([]);
  const [chartData, setChartData] = useState<any[]>([]);

  useEffect(() => {
    // Mock data - 后续替换为 API 调用
    setStats({ total: 156, critical: 12, high: 45, medium: 99 });
    
    setRecentEvents([
      { id: 1, rule: '重入攻击检测', severity: 'critical', tx_hash: '0xabc...123', time: '2分钟前' },
      { id: 2, rule: '大额转账', severity: 'high', tx_hash: '0xdef...456', time: '5分钟前' },
      { id: 3, rule: '异常Gas消耗', severity: 'medium', tx_hash: '0x789...abc', time: '10分钟前' },
    ]);

    setChartData([
      { time: '00:00', events: 4 },
      { time: '04:00', events: 3 },
      { time: '08:00', events: 8 },
      { time: '12:00', events: 12 },
      { time: '16:00', events: 7 },
      { time: '20:00', events: 5 },
    ]);
  }, []);

  const columns = [
    { title: '规则', dataIndex: 'rule', key: 'rule' },
    { title: '严重程度', dataIndex: 'severity', key: 'severity',
      render: (severity: string) => {
        const colors: any = { critical: 'red', high: 'orange', medium: 'blue' };
        return <span style={{ color: colors[severity] }}>{severity.toUpperCase()}</span>;
      }
    },
    { title: '交易哈希', dataIndex: 'tx_hash', key: 'tx_hash' },
    { title: '时间', dataIndex: 'time', key: 'time' },
  ];

  return (
    <div>
      <h1>仪表板</h1>
      
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic title="总风险事件" value={stats.total} prefix={<AlertOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="严重" value={stats.critical} valueStyle={{ color: '#cf1322' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="高危" value={stats.high} valueStyle={{ color: '#fa8c16' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="中危" value={stats.medium} valueStyle={{ color: '#1890ff' }} />
          </Card>
        </Col>
      </Row>

      <Card title="24小时风险趋势" style={{ marginBottom: 24 }}>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Line type="monotone" dataKey="events" stroke="#8884d8" name="风险事件" />
          </LineChart>
        </ResponsiveContainer>
      </Card>

      <Card title="最近风险事件">
        <Table columns={columns} dataSource={recentEvents} rowKey="id" pagination={false} />
      </Card>
    </div>
  );
};

export default Dashboard;
