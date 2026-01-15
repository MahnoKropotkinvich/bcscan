import React, { useEffect, useState } from 'react';
import { Table, Tag, Card, Input, Select } from 'antd';

const { Search } = Input;
const { Option } = Select;

interface RiskEvent {
  id: number;
  event_type: string;
  severity: string;
  tx_hash: string;
  score: number;
  detected_at: string;
  description: string;
}

const RiskEvents: React.FC = () => {
  const [events, setEvents] = useState<RiskEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>('all');

  useEffect(() => {
    // Mock data - 后续替换为 API 调用
    setEvents([
      {
        id: 1,
        event_type: 'reentrancy-attack-detection',
        severity: 'critical',
        tx_hash: '0xabc123def456...',
        score: 95,
        detected_at: '2026-01-15 19:30:00',
        description: '检测到疑似重入攻击'
      },
      {
        id: 2,
        event_type: 'large-value-transfer',
        severity: 'high',
        tx_hash: '0xdef456abc789...',
        score: 75,
        detected_at: '2026-01-15 19:25:00',
        description: '检测到大额转账'
      },
      {
        id: 3,
        event_type: 'reentrancy-attack-detection',
        severity: 'critical',
        tx_hash: '0x789abc123def...',
        score: 100,
        detected_at: '2026-01-15 19:20:00',
        description: '检测到疑似重入攻击'
      },
    ]);
  }, []);

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '事件类型',
      dataIndex: 'event_type',
      key: 'event_type',
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => {
        const colors: any = {
          critical: 'red',
          high: 'orange',
          medium: 'blue',
          low: 'green'
        };
        return <Tag color={colors[severity]}>{severity.toUpperCase()}</Tag>;
      },
    },
    {
      title: '风险分数',
      dataIndex: 'score',
      key: 'score',
      render: (score: number) => {
        const color = score >= 80 ? 'red' : score >= 60 ? 'orange' : 'blue';
        return <span style={{ color, fontWeight: 'bold' }}>{score}</span>;
      },
    },
    {
      title: '交易哈希',
      dataIndex: 'tx_hash',
      key: 'tx_hash',
      render: (hash: string) => (
        <a href={`https://etherscan.io/tx/${hash}`} target="_blank" rel="noopener noreferrer">
          {hash}
        </a>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '检测时间',
      dataIndex: 'detected_at',
      key: 'detected_at',
    },
  ];

  const filteredEvents = filter === 'all' 
    ? events 
    : events.filter(e => e.severity === filter);

  return (
    <div>
      <h1>风险事件</h1>
      
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 16 }}>
          <Search
            placeholder="搜索交易哈希"
            style={{ width: 300 }}
            onSearch={(value) => console.log(value)}
          />
          <Select
            defaultValue="all"
            style={{ width: 150 }}
            onChange={(value) => setFilter(value)}
          >
            <Option value="all">全部</Option>
            <Option value="critical">严重</Option>
            <Option value="high">高危</Option>
            <Option value="medium">中危</Option>
            <Option value="low">低危</Option>
          </Select>
        </div>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={filteredEvents}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>
    </div>
  );
};

export default RiskEvents;
