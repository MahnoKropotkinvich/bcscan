import React, { useEffect, useState, useCallback } from 'react';
import { Table, Tag, Card, Input, Select, Spin, message, Pagination } from 'antd';
import { getRiskEvents, wsClient, RiskEvent } from '../api';

const { Search } = Input;

const RiskEvents: React.FC = () => {
  const [events, setEvents] = useState<RiskEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>('');
  const [search, setSearch] = useState<string>('');
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [total, setTotal] = useState(0);

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (filter && filter !== 'all') params.severity = filter;
      if (search) params.search = search;

      const res = await getRiskEvents(params);
      setEvents(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (err) {
      console.error('Failed to fetch risk events:', err);
      message.error('获取风险事件失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filter, search]);

  useEffect(() => {
    fetchEvents();
  }, [fetchEvents]);

  // WebSocket 实时更新：收到新事件后静默刷新列表
  useEffect(() => {
    wsClient.connect();
    const unsubscribe = wsClient.onMessage(() => {
      // 静默重新获取当前页数据，避免手动拼接导致的重复问题
      fetchEvents();
    });
    return () => {
      unsubscribe();
    };
  }, []); // 空依赖，只订阅一次

  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handleFilterChange = (value: string) => {
    setFilter(value);
    setPage(1);
  };

  const severityColors: Record<string, string> = {
    critical: 'red',
    high: 'orange',
    medium: 'blue',
    low: 'green',
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '事件类型',
      dataIndex: 'event_type',
      key: 'event_type',
      width: 200,
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severityColors[severity] || 'default'}>{severity.toUpperCase()}</Tag>
      ),
    },
    {
      title: '风险分数',
      dataIndex: 'score',
      key: 'score',
      width: 90,
      render: (score: number) => {
        const color = score >= 80 ? '#cf1322' : score >= 60 ? '#fa8c16' : '#1890ff';
        return <span style={{ color, fontWeight: 'bold' }}>{score.toFixed(1)}</span>;
      },
    },
    {
      title: '交易哈希',
      dataIndex: 'tx_hash',
      key: 'tx_hash',
      ellipsis: true,
      render: (hash: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: '12px' }} title={hash}>
          {hash}
        </span>
      ),
    },
    {
      title: '合约地址',
      dataIndex: 'contract_address',
      key: 'contract_address',
      ellipsis: true,
      render: (addr: string) =>
        addr ? (
          <span style={{ fontFamily: 'monospace', fontSize: '12px' }} title={addr}>
            {addr}
          </span>
        ) : (
          '-'
        ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '检测时间',
      dataIndex: 'detected_at',
      key: 'detected_at',
      width: 170,
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
  ];

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>风险事件</h2>

      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          <Search
            placeholder="搜索交易哈希、合约地址或描述"
            style={{ width: 400 }}
            onSearch={handleSearch}
            allowClear
            enterButton
          />
          <Select
            defaultValue="all"
            style={{ width: 150 }}
            onChange={handleFilterChange}
          >
            <Select.Option value="all">全部</Select.Option>
            <Select.Option value="critical">严重</Select.Option>
            <Select.Option value="high">高危</Select.Option>
            <Select.Option value="medium">中危</Select.Option>
            <Select.Option value="low">低危</Select.Option>
          </Select>
          <span style={{ color: '#999' }}>共 {total} 条记录</span>
        </div>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={events}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="small"
          scroll={{ x: 1200 }}
        />
        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Pagination
            current={page}
            total={total}
            pageSize={pageSize}
            showSizeChanger={false}
            showQuickJumper
            showTotal={(total) => `共 ${total} 条`}
            onChange={(p) => setPage(p)}
          />
        </div>
      </Card>
    </div>
  );
};

export default RiskEvents;
