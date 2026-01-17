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

const EVENT_TYPES = [
  { type: 'reentrancy-attack', desc: '检测到疑似重入攻击', severity: ['critical', 'high'] },
  { type: 'large-value-transfer', desc: '检测到大额转账', severity: ['high', 'medium'] },
  { type: 'abnormal-gas', desc: '异常Gas消耗', severity: ['medium', 'high'] },
  { type: 'permission-abuse', desc: '权限滥用检测', severity: ['critical', 'high'] },
];

const generateTxHash = () => '0x' + Math.random().toString(16).substr(2, 40);
const formatTime = (date: Date) => date.toISOString().replace('T', ' ').substr(0, 19);

const RiskEvents: React.FC = () => {
  const [events, setEvents] = useState<RiskEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>('all');

  useEffect(() => {
    // 初始化数据
    const initialEvents = Array.from({ length: 8 }, (_, i) => {
      const eventType = EVENT_TYPES[Math.floor(Math.random() * EVENT_TYPES.length)];
      const severity = eventType.severity[Math.floor(Math.random() * eventType.severity.length)];
      const baseScore = severity === 'critical' ? 85 : severity === 'high' ? 65 : 45;
      
      return {
        id: 1000 + i,
        event_type: eventType.type,
        severity,
        tx_hash: generateTxHash(),
        score: baseScore + Math.floor(Math.random() * 15),
        detected_at: formatTime(new Date(Date.now() - Math.random() * 3600000)),
        description: eventType.desc,
      };
    });
    setEvents(initialEvents.sort((a, b) => b.id - a.id));

    const interval = setInterval(() => {
      const eventType = EVENT_TYPES[Math.floor(Math.random() * EVENT_TYPES.length)];
      const severity = eventType.severity[Math.floor(Math.random() * eventType.severity.length)];
      const baseScore = severity === 'critical' ? 85 : severity === 'high' ? 65 : 45;

      setEvents(prev => {
        const newEvent = {
          id: Date.now(),
          event_type: eventType.type,
          severity,
          tx_hash: generateTxHash(),
          score: baseScore + Math.floor(Math.random() * 15),
          detected_at: formatTime(new Date()),
          description: eventType.desc,
        };
        console.log('[RiskEvents] 新事件:', newEvent.event_type, newEvent.severity);
        return [newEvent, ...prev.slice(0, 19)];
      });
    }, 5000);

    return () => clearInterval(interval);
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
