import React, { useEffect, useState, useCallback } from 'react';
import {
  Card, Table, Tag, Button, Select, Input, message, Spin,
  Row, Col, Statistic, Modal, Descriptions, Timeline, Space, Badge, Tooltip,
} from 'antd';
import {
  AlertOutlined, BellOutlined, CheckCircleOutlined, CloseCircleOutlined,
  ExclamationCircleOutlined, EyeOutlined, ClockCircleOutlined,
  LinkOutlined, BlockOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import {
  getAlerts, getAlertStats, getAlertDetail, acknowledgeAlert,
  resolveAlert, ignoreAlert, addAlertNote, wsClient,
  Alert, AlertStats, AlertDetail, AlertHistory,
} from '../api';

const { Search } = Input;
const { TextArea } = Input;

const severityColors: Record<string, string> = {
  critical: 'red', high: 'orange', medium: 'blue', low: 'green',
};

const statusConfig: Record<string, { color: string; label: string; icon: React.ReactNode }> = {
  pending:      { color: 'warning',    label: '待处理', icon: <ClockCircleOutlined /> },
  acknowledged: { color: 'processing', label: '已确认', icon: <EyeOutlined /> },
  resolved:     { color: 'success',    label: '已解决', icon: <CheckCircleOutlined /> },
  ignored:      { color: 'default',    label: '已忽略', icon: <CloseCircleOutlined /> },
};

const actionLabel = (a: string) => {
  const map: Record<string, string> = {
    created: '创建', acknowledged: '确认', resolved: '解决',
    ignored: '忽略', note_added: '添加备注', reopened: '重新打开',
  };
  return map[a] || a;
};

const shortenHash = (hash: string) => {
  if (!hash || hash.length < 16) return hash || '-';
  return `${hash.slice(0, 10)}...${hash.slice(-6)}`;
};

const AlertCenter: React.FC = () => {
  const navigate = useNavigate();

  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [stats, setStats] = useState<AlertStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [severityFilter, setSeverityFilter] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 15;

  // 详情 modal
  const [detailVisible, setDetailVisible] = useState(false);
  const [alertDetail, setAlertDetail] = useState<AlertDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // 处理 modal
  const [actionModalVisible, setActionModalVisible] = useState(false);
  const [actionType, setActionType] = useState<'resolve' | 'ignore'>('resolve');
  const [actionAlertId, setActionAlertId] = useState<number>(0);
  const [actionNote, setActionNote] = useState('');

  const fetchAlerts = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (statusFilter && statusFilter !== 'all') params.status = statusFilter;
      if (severityFilter && severityFilter !== 'all') params.severity = severityFilter;
      if (search) params.search = search;

      const res = await getAlerts(params);
      setAlerts(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch {
      message.error('获取告警列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter, severityFilter, search]);

  const fetchStats = useCallback(async () => {
    try {
      const res = await getAlertStats();
      if (res.data.success) {
        setStats(res.data.data);
      }
    } catch { /* ignore */ }
  }, []);

  useEffect(() => { fetchAlerts(); }, [fetchAlerts]);
  useEffect(() => { fetchStats(); }, [fetchStats]);

  // WebSocket 实时更新
  useEffect(() => {
    wsClient.connect();
    const unsubscribe = wsClient.onMessage(() => {
      fetchAlerts();
      fetchStats();
    });
    return () => { unsubscribe(); };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSearch = (value: string) => { setSearch(value); setPage(1); };

  const showDetail = async (id: number) => {
    setDetailLoading(true);
    setDetailVisible(true);
    try {
      const res = await getAlertDetail(id);
      if (res.data.success) {
        setAlertDetail(res.data.data);
      }
    } catch {
      message.error('获取告警详情失败');
    } finally {
      setDetailLoading(false);
    }
  };

  const handleAcknowledge = async (id: number) => {
    try {
      await acknowledgeAlert(id);
      message.success('告警已确认');
      fetchAlerts();
      fetchStats();
    } catch (err: any) {
      message.error(err.response?.data?.error || '操作失败');
    }
  };

  const openActionModal = (id: number, type: 'resolve' | 'ignore') => {
    setActionAlertId(id);
    setActionType(type);
    setActionNote('');
    setActionModalVisible(true);
  };

  const handleAction = async () => {
    try {
      if (actionType === 'resolve') {
        await resolveAlert(actionAlertId, actionNote);
        message.success('告警已解决');
      } else {
        await ignoreAlert(actionAlertId, actionNote);
        message.success('告警已忽略');
      }
      setActionModalVisible(false);
      fetchAlerts();
      fetchStats();
    } catch (err: any) {
      message.error(err.response?.data?.error || '操作失败');
    }
  };

  // 跳转交易浏览器
  const goToTx = (hash: string) => {
    if (hash) navigate(`/explorer?tx=${hash}`);
  };

  // 跳转地址
  const goToAddress = (addr: string) => {
    if (addr) navigate(`/explorer?addr=${addr}`);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 55 },
    {
      title: '告警标题', dataIndex: 'title', key: 'title',
      ellipsis: true,
      render: (title: string) => <strong>{title}</strong>,
    },
    {
      title: '严重程度', dataIndex: 'severity', key: 'severity', width: 90,
      render: (s: string) => <Tag color={severityColors[s]}>{s.toUpperCase()}</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) => {
        const cfg = statusConfig[s] || { color: 'default', label: s, icon: null };
        return <Badge status={cfg.color as any} text={cfg.label} />;
      },
    },
    {
      title: '风险分数', dataIndex: 'score', key: 'score', width: 80,
      render: (score: number) => {
        if (!score) return '-';
        const color = score >= 80 ? '#cf1322' : score >= 60 ? '#fa8c16' : '#1890ff';
        return <span style={{ color, fontWeight: 'bold' }}>{score.toFixed(1)}</span>;
      },
    },
    {
      title: '交易哈希', dataIndex: 'tx_hash', key: 'tx_hash', width: 180,
      render: (hash: string) => hash ? (
        <Tooltip title={hash}>
          <Button type="link" size="small" onClick={() => goToTx(hash)}
            style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
            <BlockOutlined style={{ marginRight: 4, fontSize: 11 }} />
            {shortenHash(hash)}
          </Button>
        </Tooltip>
      ) : '-',
    },
    {
      title: '合约地址', dataIndex: 'contract_address', key: 'contract_address', width: 170,
      render: (addr: string) => addr ? (
        <Tooltip title={addr}>
          <Button type="link" size="small" onClick={() => goToAddress(addr)}
            style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
            <LinkOutlined style={{ marginRight: 4, fontSize: 11 }} />
            {shortenHash(addr)}
          </Button>
        </Tooltip>
      ) : '-',
    },
    {
      title: '时间', dataIndex: 'created_at', key: 'created_at', width: 155,
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
    {
      title: '操作', key: 'action', width: 190, fixed: 'right' as const,
      render: (_: any, record: Alert) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => showDetail(record.id)}>详情</Button>
          {record.status === 'pending' && (
            <Button type="link" size="small" onClick={() => handleAcknowledge(record.id)}>确认</Button>
          )}
          {(record.status === 'pending' || record.status === 'acknowledged') && (
            <>
              <Button type="link" size="small" style={{ color: '#52c41a' }}
                onClick={() => openActionModal(record.id, 'resolve')}>解决</Button>
              <Button type="link" size="small" style={{ color: '#999' }}
                onClick={() => openActionModal(record.id, 'ignore')}>忽略</Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>
        <AlertOutlined style={{ marginRight: 8 }} />
        告警中心
      </h2>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic title="总告警" value={stats?.total || 0} prefix={<BellOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="待处理" value={stats?.pending_count || 0}
              valueStyle={{ color: '#fa8c16' }} prefix={<ExclamationCircleOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="已确认" value={stats?.by_status?.acknowledged || 0}
              valueStyle={{ color: '#1890ff' }} prefix={<EyeOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="已处置" value={(stats?.by_status?.resolved || 0) + (stats?.by_status?.ignored || 0)}
              valueStyle={{ color: '#52c41a' }} prefix={<CheckCircleOutlined />} />
          </Card>
        </Col>
      </Row>

      {/* 筛选 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 16, alignItems: 'center', flexWrap: 'wrap' }}>
          <Search placeholder="搜索告警标题、交易哈希或合约地址" style={{ width: 380 }}
            onSearch={handleSearch} allowClear enterButton />
          <Select defaultValue="all" style={{ width: 130 }} onChange={(v) => { setStatusFilter(v); setPage(1); }}>
            <Select.Option value="all">全部状态</Select.Option>
            <Select.Option value="pending">待处理</Select.Option>
            <Select.Option value="acknowledged">已确认</Select.Option>
            <Select.Option value="resolved">已解决</Select.Option>
            <Select.Option value="ignored">已忽略</Select.Option>
          </Select>
          <Select defaultValue="all" style={{ width: 130 }} onChange={(v) => { setSeverityFilter(v); setPage(1); }}>
            <Select.Option value="all">全部等级</Select.Option>
            <Select.Option value="critical">严重</Select.Option>
            <Select.Option value="high">高危</Select.Option>
            <Select.Option value="medium">中危</Select.Option>
            <Select.Option value="low">低危</Select.Option>
          </Select>
          <span style={{ color: '#999' }}>共 {total} 条</span>
        </div>
      </Card>

      {/* 告警列表 */}
      <Card>
        <Table columns={columns} dataSource={alerts} rowKey="id" loading={loading}
          pagination={{
            current: page, total, pageSize,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p) => setPage(p),
          }}
          size="small" scroll={{ x: 1400 }} />
      </Card>

      {/* 告警详情 Modal */}
      <Modal title="告警详情" open={detailVisible} onCancel={() => setDetailVisible(false)}
        footer={null} width={750}>
        {detailLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : alertDetail ? (
          <div>
            <Descriptions bordered size="small" column={2} style={{ marginBottom: 16 }}>
              <Descriptions.Item label="告警 ID">{alertDetail.alert.id}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Badge status={(statusConfig[alertDetail.alert.status]?.color || 'default') as any}
                  text={statusConfig[alertDetail.alert.status]?.label || alertDetail.alert.status} />
              </Descriptions.Item>
              <Descriptions.Item label="标题">{alertDetail.alert.title}</Descriptions.Item>
              <Descriptions.Item label="严重程度">
                <Tag color={severityColors[alertDetail.alert.severity]}>{alertDetail.alert.severity.toUpperCase()}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>{alertDetail.alert.message || '-'}</Descriptions.Item>
              <Descriptions.Item label="交易哈希" span={2}>
                {alertDetail.alert.tx_hash ? (
                  <Button type="link" onClick={() => { setDetailVisible(false); goToTx(alertDetail.alert.tx_hash!); }}
                    style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
                    <BlockOutlined style={{ marginRight: 4 }} />
                    {alertDetail.alert.tx_hash}
                  </Button>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="合约地址" span={2}>
                {alertDetail.alert.contract_address ? (
                  <Button type="link" onClick={() => { setDetailVisible(false); goToAddress(alertDetail.alert.contract_address!); }}
                    style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
                    <LinkOutlined style={{ marginRight: 4 }} />
                    {alertDetail.alert.contract_address}
                  </Button>
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{new Date(alertDetail.alert.created_at).toLocaleString('zh-CN')}</Descriptions.Item>
              <Descriptions.Item label="风险分数">{alertDetail.alert.score?.toFixed(1) || '-'}</Descriptions.Item>
              {alertDetail.alert.notes && (
                <Descriptions.Item label="备注" span={2}>
                  <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: 13 }}>{alertDetail.alert.notes}</pre>
                </Descriptions.Item>
              )}
            </Descriptions>

            {alertDetail.history.length > 0 && (
              <Card title="处理历史" size="small">
                <Timeline
                  items={alertDetail.history.map((h: AlertHistory) => ({
                    color: h.action === 'resolved' ? 'green' : h.action === 'ignored' ? 'gray' : 'blue',
                    children: (
                      <div>
                        <strong>{actionLabel(h.action)}</strong>
                        <span style={{ color: '#999', marginLeft: 8 }}>
                          {h.username} · {new Date(h.created_at).toLocaleString('zh-CN')}
                        </span>
                        {h.old_status && h.new_status && (
                          <div style={{ fontSize: 12, color: '#666' }}>
                            {h.old_status} → {h.new_status}
                          </div>
                        )}
                        {h.note && <div style={{ fontSize: 13, marginTop: 4 }}>{h.note}</div>}
                      </div>
                    ),
                  }))}
                />
              </Card>
            )}
          </div>
        ) : null}
      </Modal>

      {/* 操作 Modal */}
      <Modal
        title={actionType === 'resolve' ? '解决告警' : '忽略告警'}
        open={actionModalVisible}
        onOk={handleAction}
        onCancel={() => setActionModalVisible(false)}
      >
        <p>{actionType === 'resolve' ? '请填写解决方案说明（可选）：' : '请填写忽略原因（可选）：'}</p>
        <TextArea rows={3} value={actionNote} onChange={(e) => setActionNote(e.target.value)}
          placeholder={actionType === 'resolve' ? '描述如何解决该告警...' : '描述忽略原因...'} />
      </Modal>
    </div>
  );
};

export default AlertCenter;
