import React, { useEffect, useState, useCallback } from 'react';
import { Card, Table, Tag, Select, Input, message, Pagination } from 'antd';
import { AuditOutlined, UserOutlined } from '@ant-design/icons';
import { getAuditLogs, getAuditActions, AuditLogEntry } from '../api';

const { Search } = Input;

// 操作类型标签和颜色
const ACTION_CONFIG: Record<string, { label: string; color: string }> = {
  login_success: { label: '登录成功', color: 'green' },
  login_failed: { label: '登录失败', color: 'red' },
  user_register: { label: '用户注册', color: 'blue' },
  rule_reload: { label: '规则热加载', color: 'purple' },
  user_role_change: { label: '角色修改', color: 'orange' },
  user_status_change: { label: '状态修改', color: 'orange' },
  alert_acknowledge: { label: '告警确认', color: 'cyan' },
  alert_resolve: { label: '告警解决', color: 'green' },
  alert_ignore: { label: '告警忽略', color: 'default' },
};

const AuditLog: React.FC = () => {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [actionFilter, setActionFilter] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [actions, setActions] = useState<string[]>([]);
  const pageSize = 20;

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (actionFilter && actionFilter !== 'all') params.action = actionFilter;
      if (search) params.search = search;

      const res = await getAuditLogs(params);
      if (res.data.success) {
        setLogs(res.data.data.items || []);
        setTotal(res.data.data.total || 0);
      }
    } catch {
      message.error('获取审计日志失败');
    } finally {
      setLoading(false);
    }
  }, [page, actionFilter, search]);

  const fetchActions = useCallback(async () => {
    try {
      const res = await getAuditActions();
      if (res.data.success) {
        setActions(res.data.data || []);
      }
    } catch { /* ignore */ }
  }, []);

  useEffect(() => { fetchLogs(); }, [fetchLogs]);
  useEffect(() => { fetchActions(); }, [fetchActions]);

  const handleSearch = (value: string) => { setSearch(value); setPage(1); };

  const parseDetails = (details: string): Record<string, any> | null => {
    if (!details) return null;
    try {
      return JSON.parse(details);
    } catch {
      return null;
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    {
      title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170,
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
    {
      title: '用户', dataIndex: 'username', key: 'username', width: 120,
      render: (name: string) => (
        <span><UserOutlined style={{ marginRight: 4 }} />{name}</span>
      ),
    },
    {
      title: '操作', dataIndex: 'action', key: 'action', width: 130,
      render: (action: string) => {
        const cfg = ACTION_CONFIG[action] || { label: action, color: 'default' };
        return <Tag color={cfg.color}>{cfg.label}</Tag>;
      },
    },
    {
      title: '资源', dataIndex: 'resource', key: 'resource', width: 200,
      ellipsis: true,
      render: (resource: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{resource || '-'}</span>
      ),
    },
    {
      title: '详情', dataIndex: 'details', key: 'details',
      ellipsis: true,
      render: (details: string) => {
        const parsed = parseDetails(details);
        if (!parsed) return <span style={{ color: '#999' }}>-</span>;
        return (
          <span style={{ fontSize: 12, color: '#666' }}>
            {Object.entries(parsed).map(([k, v]) => `${k}=${v}`).join(', ')}
          </span>
        );
      },
    },
    {
      title: 'IP 地址', dataIndex: 'ip_address', key: 'ip_address', width: 130,
      render: (ip: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{ip || '-'}</span>
      ),
    },
  ];

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>
        <AuditOutlined style={{ marginRight: 8 }} />
        审计日志
      </h2>

      {/* 筛选 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          <Search
            placeholder="搜索用户名、资源或详情"
            style={{ width: 350 }}
            onSearch={handleSearch}
            allowClear
            enterButton
          />
          <Select
            defaultValue="all"
            style={{ width: 160 }}
            onChange={(v) => { setActionFilter(v); setPage(1); }}
          >
            <Select.Option value="all">全部操作</Select.Option>
            {actions.map((a) => (
              <Select.Option key={a} value={a}>
                {ACTION_CONFIG[a]?.label || a}
              </Select.Option>
            ))}
          </Select>
          <span style={{ color: '#999' }}>共 {total} 条记录</span>
        </div>
      </Card>

      {/* 日志列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={false}
          size="small"
          scroll={{ x: 1100 }}
        />
        <div style={{ marginTop: 16, textAlign: 'right' }}>
          <Pagination
            current={page}
            total={total}
            pageSize={pageSize}
            showQuickJumper
            showTotal={(t) => `共 ${t} 条`}
            onChange={(p) => setPage(p)}
          />
        </div>
      </Card>
    </div>
  );
};

export default AuditLog;
