import React, { useEffect, useState, useCallback } from 'react';
import {
  Card, Input, Table, Tag, Descriptions, Spin, Row, Col,
  Button, Space, message, Badge, Collapse, Typography, Empty, Tooltip,
} from 'antd';
import {
  SearchOutlined, SwapOutlined, CodeOutlined, LinkOutlined,
  WarningOutlined, BlockOutlined, ArrowRightOutlined,
  FunctionOutlined, BranchesOutlined, ApiOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { useSearchParams, useNavigate } from 'react-router-dom';
import {
  getTransactionByHash, getTransactionsByAddress, getAddressSummary,
  getRisksByTxHash, getRecentTransactions, getAlertsByTxHash,
  ExplorerTransaction, TxBrief, AddressSummary, RiskEvent, CallFrame, Alert,
} from '../api';

const { Search } = Input;
const { Text } = Typography;

// ==================== 调用栈树状可视化组件 ====================

const CALL_TYPE_STYLES: Record<string, { color: string; bg: string; border: string }> = {
  CALL:         { color: '#1677ff', bg: '#e6f4ff', border: '#91caff' },
  DELEGATECALL: { color: '#d46b08', bg: '#fff7e6', border: '#ffd591' },
  STATICCALL:   { color: '#389e0d', bg: '#f6ffed', border: '#b7eb8f' },
  CREATE:       { color: '#cf1322', bg: '#fff1f0', border: '#ffa39e' },
  CREATE2:      { color: '#cf1322', bg: '#fff1f0', border: '#ffa39e' },
};

const shortenAddr = (addr: string) => {
  if (!addr || addr.length < 12) return addr || '';
  return `${addr.slice(0, 8)}...${addr.slice(-6)}`;
};

const weiToEth = (wei: string) => {
  if (!wei || wei === '' || wei === '0' || wei === '0x0') return '';
  try {
    const val = BigInt(wei.startsWith('0x') ? wei : wei);
    const eth = Number(val) / 1e18;
    if (eth > 0.0001) return `${eth.toFixed(4)} ETH`;
    if (val > BigInt(0)) return `${val.toString()} wei`;
    return '';
  } catch { return ''; }
};

interface CallTraceNodeProps {
  frame: CallFrame;
  index: number;
  onAddressClick?: (addr: string) => void;
}

const CallTraceNode: React.FC<CallTraceNodeProps> = ({ frame, index, onAddressClick }) => {
  const style = CALL_TYPE_STYLES[frame.type] || CALL_TYPE_STYLES.CALL;
  const indent = frame.depth * 28;
  const hasError = !!frame.error;
  const valueStr = weiToEth(frame.value);

  const funcDisplay = frame.function_name
    ? frame.function_name
    : frame.function && frame.function.length >= 10
      ? frame.function
      : null;

  return (
    <div
      style={{
        marginLeft: indent,
        marginBottom: 2,
        padding: '6px 10px',
        borderLeft: `3px solid ${hasError ? '#ff4d4f' : style.border}`,
        background: hasError ? '#fff2f0' : (index % 2 === 0 ? '#fafafa' : '#fff'),
        borderRadius: '0 4px 4px 0',
        fontSize: 13,
        fontFamily: "'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace",
        transition: 'background 0.2s',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
        <span style={{ color: '#bbb', fontSize: 11, minWidth: 20 }}>#{index}</span>
        <Tag
          style={{
            color: style.color, background: style.bg, border: `1px solid ${style.border}`,
            fontWeight: 600, fontSize: 11, lineHeight: '18px', padding: '0 6px',
          }}
        >
          {frame.type || 'CALL'}
        </Tag>
        {funcDisplay && (
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
            <FunctionOutlined style={{ color: '#722ed1', fontSize: 12 }} />
            {frame.function_name ? (
              <Tooltip title={frame.function_desc || frame.function || ''}>
                <Tag color="purple" style={{ margin: 0, fontWeight: 600 }}>
                  {frame.function_name}()
                </Tag>
              </Tooltip>
            ) : (
              <Text code style={{ fontSize: 11, color: '#999' }}>{frame.function}</Text>
            )}
          </span>
        )}
        {hasError && <Tag color="error" style={{ margin: 0 }}>{frame.error}</Tag>}
        {valueStr && (
          <span style={{ color: '#cf1322', fontWeight: 500, fontSize: 12 }}>
            [{valueStr}]
          </span>
        )}
        {frame.gas_used > 0 && (
          <span style={{ color: '#999', fontSize: 11 }}>
            gas: {frame.gas_used.toLocaleString()}
          </span>
        )}
      </div>
      <div style={{ marginTop: 3, display: 'flex', alignItems: 'center', gap: 4, fontSize: 12 }}>
        {frame.from && (
          <Tooltip title={frame.from}>
            <Button type="link" size="small"
              onClick={() => onAddressClick?.(frame.from)}
              style={{ padding: 0, fontSize: 12, fontFamily: 'monospace', height: 'auto', color: '#1677ff' }}>
              {shortenAddr(frame.from)}
            </Button>
          </Tooltip>
        )}
        <ArrowRightOutlined style={{ color: '#ccc', fontSize: 10 }} />
        {frame.to ? (
          <Tooltip title={frame.to}>
            <Button type="link" size="small"
              onClick={() => onAddressClick?.(frame.to)}
              style={{ padding: 0, fontSize: 12, fontFamily: 'monospace', height: 'auto', color: '#1677ff' }}>
              {shortenAddr(frame.to)}
            </Button>
          </Tooltip>
        ) : (
          <Text type="secondary" style={{ fontSize: 12 }}>[Contract Creation]</Text>
        )}
        {frame.function_desc && (
          <Text type="secondary" style={{ fontSize: 11, marginLeft: 8 }}>
            — {frame.function_desc}
          </Text>
        )}
      </div>
    </div>
  );
};

interface CallTraceTreeProps {
  frames: CallFrame[];
  onAddressClick?: (addr: string) => void;
}

const CallTraceTree: React.FC<CallTraceTreeProps> = ({ frames, onAddressClick }) => {
  const maxDepth = Math.max(0, ...frames.map(f => f.depth));
  const callTypes = new Set(frames.map(f => f.type).filter(Boolean));
  const errorCount = frames.filter(f => f.error).length;
  const namedFunctions = frames.filter(f => f.function_name).length;

  return (
    <div>
      <div style={{
        display: 'flex', gap: 16, marginBottom: 12, padding: '8px 12px',
        background: '#f5f5f5', borderRadius: 4, fontSize: 12, color: '#666',
      }}>
        <span><BranchesOutlined /> {frames.length} 个调用帧</span>
        <span>最大深度: {maxDepth}</span>
        <span>调用类型: {Array.from(callTypes).map(t => (
          <Tag key={t} style={{ fontSize: 10, lineHeight: '16px', padding: '0 4px', margin: '0 2px' }}
            color={CALL_TYPE_STYLES[t]?.color}>
            {t}
          </Tag>
        ))}</span>
        {namedFunctions > 0 && <span><FunctionOutlined /> 已识别函数: {namedFunctions}/{frames.length}</span>}
        {errorCount > 0 && <span style={{ color: '#cf1322' }}>Errors: {errorCount}</span>}
      </div>
      <div style={{
        border: '1px solid #f0f0f0', borderRadius: 4, padding: '8px 8px',
        maxHeight: 500, overflow: 'auto', background: '#fff',
      }}>
        {frames.map((frame, i) => (
          <CallTraceNode key={i} frame={frame} index={i} onAddressClick={onAddressClick} />
        ))}
      </div>
    </div>
  );
};

// ==================== 交易详情视图 ====================

interface TxDetailProps {
  tx: ExplorerTransaction;
  risks: RiskEvent[];
  alerts: Alert[];
  onAddressClick?: (addr: string) => void;
  onAlertClick?: (id: number) => void;
}

const severityColors: Record<string, string> = {
  critical: 'red', high: 'orange', medium: 'blue', low: 'green',
};

const statusColors: Record<string, string> = {
  pending: 'warning', acknowledged: 'processing', resolved: 'success', ignored: 'default',
};
const statusLabels: Record<string, string> = {
  pending: '待处理', acknowledged: '已确认', resolved: '已解决', ignored: '已忽略',
};

const TxDetailView: React.FC<TxDetailProps> = ({ tx, risks, alerts, onAddressClick, onAlertClick }) => {
  return (
    <div>
      {/* 基本信息 */}
      <Card title="交易信息" size="small" style={{ marginBottom: 16 }}>
        <Descriptions bordered size="small" column={2}>
          <Descriptions.Item label="交易哈希" span={2}>
            <Text copyable style={{ fontFamily: 'monospace', fontSize: 12 }}>{tx.tx_hash}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="区块号">{tx.block_number}</Descriptions.Item>
          <Descriptions.Item label="状态">
            {tx.status === 1 ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="发送方">
            <Button type="link" onClick={() => onAddressClick?.(tx.from_address)}
              style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
              {tx.from_address}
            </Button>
          </Descriptions.Item>
          <Descriptions.Item label="接收方">
            {tx.to_address ? (
              <Button type="link" onClick={() => onAddressClick?.(tx.to_address)}
                style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
                {tx.to_address}
              </Button>
            ) : <Tag color="red">合约创建</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="金额">{weiToEth(tx.value) || '0 ETH'}</Descriptions.Item>
          <Descriptions.Item label="时间">{new Date(tx.timestamp).toLocaleString('zh-CN')}</Descriptions.Item>
          <Descriptions.Item label="Gas Used">{tx.gas_used?.toLocaleString()}</Descriptions.Item>
          <Descriptions.Item label="Gas Limit">{tx.gas_limit?.toLocaleString() || '-'}</Descriptions.Item>
          {tx.function_selector && (
            <Descriptions.Item label="函数调用" span={2}>
              <Space size="middle">
                <span>
                  <ApiOutlined style={{ marginRight: 4, color: '#722ed1' }} />
                  <Text code style={{ fontSize: 13 }}>{tx.function_selector}</Text>
                </span>
                {tx.function_name && (
                  <Tag color="purple" style={{ fontSize: 13, padding: '2px 8px' }}>
                    <FunctionOutlined style={{ marginRight: 4 }} />
                    {tx.function_name}()
                  </Tag>
                )}
                {tx.function_desc && (
                  <Text type="secondary">{tx.function_desc}</Text>
                )}
              </Space>
            </Descriptions.Item>
          )}
        </Descriptions>
      </Card>

      {/* 关联告警 */}
      {alerts.length > 0 && (
        <Card title={<span><BellOutlined style={{ color: '#fa8c16', marginRight: 8 }} />关联告警 ({alerts.length})</span>}
          size="small" style={{ marginBottom: 16 }}>
          <Table
            size="small" pagination={false} rowKey="id"
            dataSource={alerts}
            columns={[
              { title: '告警标题', dataIndex: 'title', key: 'title', ellipsis: true },
              {
                title: '等级', dataIndex: 'severity', key: 'severity', width: 80,
                render: (s: string) => <Tag color={severityColors[s]}>{s.toUpperCase()}</Tag>,
              },
              {
                title: '状态', dataIndex: 'status', key: 'status', width: 90,
                render: (s: string) => <Badge status={statusColors[s] as any} text={statusLabels[s] || s} />,
              },
              {
                title: '分数', dataIndex: 'score', key: 'score', width: 70,
                render: (s: number) => <span style={{ color: s >= 80 ? '#cf1322' : s >= 60 ? '#fa8c16' : '#1890ff', fontWeight: 'bold' }}>{s?.toFixed(1)}</span>,
              },
              {
                title: '操作', key: 'op', width: 60,
                render: (_: any, r: Alert) => (
                  <Button type="link" size="small" onClick={() => onAlertClick?.(r.id)}>
                    查看
                  </Button>
                ),
              },
            ]}
          />
        </Card>
      )}

      {/* 风险事件 */}
      {risks.length > 0 && (
        <Card title={<span><WarningOutlined style={{ color: '#cf1322', marginRight: 8 }} />风险检测记录 ({risks.length})</span>}
          size="small" style={{ marginBottom: 16 }}>
          <Table
            size="small" pagination={false} rowKey="id"
            dataSource={risks}
            columns={[
              { title: '事件类型', dataIndex: 'event_type', key: 'event_type' },
              {
                title: '等级', dataIndex: 'severity', key: 'severity', width: 80,
                render: (s: string) => <Tag color={severityColors[s]}>{s.toUpperCase()}</Tag>,
              },
              {
                title: '分数', dataIndex: 'score', key: 'score', width: 70,
                render: (s: number) => <span style={{ color: s >= 80 ? '#cf1322' : s >= 60 ? '#fa8c16' : '#1890ff', fontWeight: 'bold' }}>{s?.toFixed(1)}</span>,
              },
              { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
            ]}
          />
        </Card>
      )}

      {/* 调用栈 */}
      {tx.call_stack && tx.call_stack.length > 0 && (
        <Card
          title={<span><BranchesOutlined style={{ marginRight: 8 }} />Internal Transactions (Call Trace)</span>}
          size="small" style={{ marginBottom: 16 }}>
          <CallTraceTree frames={tx.call_stack} onAddressClick={onAddressClick} />
        </Card>
      )}

      {/* 事件日志 */}
      {tx.events_data && tx.events_data.length > 0 && (
        <Card title={`Event Logs (${tx.events_data.length})`} size="small" style={{ marginBottom: 16 }}>
          <Collapse items={tx.events_data.map((evt, i) => ({
            key: String(i),
            label: (
              <Space>
                <Badge count={i} style={{ backgroundColor: '#1677ff' }} />
                <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{shortenAddr(evt.address)}</Text>
                <Text type="secondary" style={{ fontSize: 11 }}>
                  {evt.topics?.length || 0} topics
                  {evt.data && evt.data !== '0x' ? ` · ${Math.max(0, (evt.data.length - 2) / 2)} bytes data` : ''}
                </Text>
              </Space>
            ),
            children: (
              <Descriptions size="small" column={1} bordered>
                <Descriptions.Item label="Address">
                  <Button type="link" onClick={() => onAddressClick?.(evt.address)}
                    style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}>
                    {evt.address}
                  </Button>
                </Descriptions.Item>
                {evt.topics?.map((t, j) => (
                  <Descriptions.Item key={j} label={<span>{j === 0 ? 'Topic [0] (signature)' : `Topic [${j}]`}</span>}>
                    <Text copyable style={{ fontFamily: 'monospace', fontSize: 11, wordBreak: 'break-all' }}>{t}</Text>
                  </Descriptions.Item>
                ))}
                {evt.data && evt.data !== '0x' && (
                  <Descriptions.Item label="Data">
                    <div style={{ fontFamily: 'monospace', fontSize: 11, wordBreak: 'break-all', maxHeight: 120, overflow: 'auto' }}>
                      {evt.data}
                    </div>
                  </Descriptions.Item>
                )}
              </Descriptions>
            ),
          }))} />
        </Card>
      )}

      {/* Input Data */}
      {tx.input_data && tx.input_data !== '0x' && tx.input_data.length > 2 && (
        <Card title="Input Data" size="small">
          <div style={{
            background: '#1e1e1e', color: '#d4d4d4', padding: 16, borderRadius: 4,
            fontFamily: "'SF Mono', 'Monaco', 'Menlo', 'Consolas', monospace",
            fontSize: 12, wordBreak: 'break-all', maxHeight: 200, overflow: 'auto', lineHeight: 1.6,
          }}>
            <span style={{ color: '#c586c0' }}>{tx.input_data.slice(0, 10)}</span>
            {tx.input_data.length > 10 && (
              <span>{tx.input_data.slice(10).match(/.{1,64}/g)?.map((chunk, i) => (
                <span key={i}>
                  {i === 0 ? '' : '\n'}
                  <span style={{ color: '#6a9955' }}>{String(i).padStart(4, ' ')}:</span>{' '}
                  <span style={{ color: '#ce9178' }}>{chunk}</span>
                </span>
              ))}</span>
            )}
          </div>
          {tx.function_name && (
            <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
              <FunctionOutlined style={{ marginRight: 4 }} />
              Method: <Text strong>{tx.function_name}()</Text>
              {tx.function_desc && <Text type="secondary"> — {tx.function_desc}</Text>}
            </div>
          )}
        </Card>
      )}
    </div>
  );
};

// ==================== 主页面 ====================

const TransactionExplorer: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const [searchValue, setSearchValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [txDetail, setTxDetail] = useState<ExplorerTransaction | null>(null);
  const [txRisks, setTxRisks] = useState<RiskEvent[]>([]);
  const [txAlerts, setTxAlerts] = useState<Alert[]>([]);
  const [addressTxs, setAddressTxs] = useState<TxBrief[]>([]);
  const [addressSummary, setAddressSummary] = useState<AddressSummary | null>(null);
  const [addressTotal, setAddressTotal] = useState(0);
  const [addressPage, setAddressPage] = useState(1);
  const [recentTxs, setRecentTxs] = useState<TxBrief[]>([]);
  const [recentTotal, setRecentTotal] = useState(0);
  const [recentPage, setRecentPage] = useState(1);
  const [viewMode, setViewMode] = useState<'home' | 'tx' | 'address'>('home');

  const fetchRecent = useCallback(async (p: number = 1) => {
    try {
      const res = await getRecentTransactions({ page: p, page_size: 15 });
      setRecentTxs(res.data.items || []);
      setRecentTotal(res.data.total || 0);
      setRecentPage(p);
    } catch { /* ignore */ }
  }, []);

  // 处理 URL 参数跳转（从告警中心等页面跳入）
  useEffect(() => {
    const txParam = searchParams.get('tx');
    const addrParam = searchParams.get('addr');
    if (txParam) {
      handleTxClick(txParam);
    } else if (addrParam) {
      handleAddressClick(addrParam);
    } else {
      fetchRecent();
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSearch = async (value: string) => {
    if (!value.trim()) return;
    const query = value.trim();
    setLoading(true);

    if (query.startsWith('0x') && query.length === 66) {
      await loadTxDetail(query);
    } else if (query.startsWith('0x') && query.length === 42) {
      await loadAddress(query);
    } else {
      message.warning('请输入有效的交易哈希(0x...66位)或地址(0x...42位)');
    }
    setLoading(false);
  };

  const loadTxDetail = async (hash: string) => {
    try {
      const [txRes, riskRes, alertRes] = await Promise.all([
        getTransactionByHash(hash),
        getRisksByTxHash(hash),
        getAlertsByTxHash(hash),
      ]);
      if (txRes.data.success) {
        setTxDetail(txRes.data.data);
        setTxRisks(riskRes.data.success ? riskRes.data.data || [] : []);
        setTxAlerts(alertRes.data.success ? alertRes.data.data || [] : []);
        setViewMode('tx');
        setSearchParams({ tx: hash }, { replace: true });
      } else {
        message.warning('未找到该交易');
      }
    } catch {
      message.warning('未找到该交易');
    }
  };

  const loadAddress = async (addr: string, page: number = 1) => {
    try {
      const [txRes, sumRes] = await Promise.all([
        getTransactionsByAddress(addr, { page, page_size: 15 }),
        getAddressSummary(addr),
      ]);
      setAddressTxs(txRes.data.items || []);
      setAddressTotal(txRes.data.total || 0);
      setAddressPage(page);
      if (sumRes.data.success) setAddressSummary(sumRes.data.data);
      setViewMode('address');
      setSearchParams({ addr }, { replace: true });
    } catch {
      message.warning('未找到该地址的交易');
    }
  };

  const handleTxClick = async (hash: string) => {
    setSearchValue(hash);
    setLoading(true);
    await loadTxDetail(hash);
    setLoading(false);
  };

  const handleAddressClick = async (addr: string) => {
    if (!addr || addr.length !== 42) return;
    setSearchValue(addr);
    setLoading(true);
    await loadAddress(addr);
    setLoading(false);
  };

  const handleAddressPageChange = async (p: number) => {
    if (!addressSummary) return;
    try {
      const res = await getTransactionsByAddress(addressSummary.address, { page: p, page_size: 15 });
      setAddressTxs(res.data.items || []);
      setAddressPage(p);
    } catch { /* ignore */ }
  };

  const goBack = () => {
    setViewMode('home');
    setSearchParams({}, { replace: true });
    fetchRecent(1);
  };

  const goToAlertCenter = (alertId?: number) => {
    navigate('/alerts');
  };

  const txColumns = [
    {
      title: '交易哈希', dataIndex: 'tx_hash', key: 'tx_hash',
      render: (hash: string) => (
        <Button type="link" size="small" onClick={() => handleTxClick(hash)}
          style={{ fontFamily: 'monospace', fontSize: 11, padding: 0 }}>
          {hash.slice(0, 14)}...{hash.slice(-8)}
        </Button>
      ),
    },
    { title: '区块', dataIndex: 'block_number', key: 'block_number', width: 80 },
    {
      title: 'From', dataIndex: 'from_address', key: 'from', ellipsis: true,
      render: (addr: string) => (
        <Button type="link" size="small" onClick={() => handleAddressClick(addr)}
          style={{ fontFamily: 'monospace', fontSize: 11, padding: 0 }}>
          {shortenAddr(addr)}
        </Button>
      ),
    },
    {
      title: '', key: 'arrow', width: 30,
      render: () => <ArrowRightOutlined style={{ color: '#ccc', fontSize: 10 }} />,
    },
    {
      title: 'To', dataIndex: 'to_address', key: 'to', ellipsis: true,
      render: (addr: string) => addr ? (
        <Button type="link" size="small" onClick={() => handleAddressClick(addr)}
          style={{ fontFamily: 'monospace', fontSize: 11, padding: 0 }}>
          {shortenAddr(addr)}
        </Button>
      ) : <Tag color="red" style={{ fontSize: 10 }}>CREATE</Tag>,
    },
    {
      title: '函数', key: 'func', width: 150,
      render: (_: any, row: TxBrief) => row.function_name ? (
        <Tag color="purple" style={{ margin: 0 }}>
          <FunctionOutlined style={{ marginRight: 2 }} />{row.function_name}
        </Tag>
      ) : row.function_selector ? (
        <Text code style={{ fontSize: 10, color: '#999' }}>{row.function_selector}</Text>
      ) : <Text type="secondary">Transfer</Text>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 60,
      render: (s: number) => s === 1 ? <Badge status="success" text="OK" /> : <Badge status="error" text="Fail" />,
    },
    {
      title: '时间', dataIndex: 'timestamp', key: 'timestamp', width: 160,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
  ];

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>
        <BlockOutlined style={{ marginRight: 8 }} />
        交易浏览器
      </h2>

      <Card style={{ marginBottom: 24 }}>
        <Search
          placeholder="输入交易哈希 (0x...66位) 或 地址 (0x...42位) 查询"
          enterButton={<span><SearchOutlined /> 查询</span>}
          size="large"
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          onSearch={handleSearch}
          loading={loading}
        />
      </Card>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" tip="查询中..." /></div>
      ) : viewMode === 'tx' && txDetail ? (
        <div>
          <Button type="link" onClick={goBack} style={{ marginBottom: 16 }}>
            ← 返回
          </Button>
          <TxDetailView tx={txDetail} risks={txRisks} alerts={txAlerts}
            onAddressClick={handleAddressClick} onAlertClick={goToAlertCenter} />
        </div>
      ) : viewMode === 'address' && addressSummary ? (
        <div>
          <Button type="link" onClick={goBack} style={{ marginBottom: 16 }}>
            ← 返回
          </Button>
          <Card title={<span><LinkOutlined style={{ marginRight: 8 }} />地址概要</span>} size="small" style={{ marginBottom: 16 }}>
            <Descriptions bordered size="small" column={4}>
              <Descriptions.Item label="地址" span={4}>
                <Text copyable style={{ fontFamily: 'monospace' }}>{addressSummary.address}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="总交易数">{addressSummary.tx_count}</Descriptions.Item>
              <Descriptions.Item label="发送">{addressSummary.sent_count}</Descriptions.Item>
              <Descriptions.Item label="接收">{addressSummary.received_count}</Descriptions.Item>
              <Descriptions.Item label="关联风险">{addressSummary.risk_count > 0 ? <span style={{ color: '#cf1322', fontWeight: 'bold' }}>{addressSummary.risk_count}</span> : 0}</Descriptions.Item>
            </Descriptions>
          </Card>

          {addressSummary.top_functions && addressSummary.top_functions.length > 0 && (
            <Card title="常调用函数 Top 5" size="small" style={{ marginBottom: 16 }}>
              <Row gutter={8}>
                {addressSummary.top_functions.map((f, i) => (
                  <Col key={i}>
                    <Tag color="blue">
                      <FunctionOutlined style={{ marginRight: 4 }} />
                      {f.name} <Badge count={f.count} size="small" style={{ marginLeft: 4 }} />
                    </Tag>
                  </Col>
                ))}
              </Row>
            </Card>
          )}

          <Card title="交易记录" size="small">
            <Table columns={txColumns} dataSource={addressTxs} rowKey="tx_hash" size="small"
              pagination={{
                current: addressPage, total: addressTotal, pageSize: 15,
                onChange: handleAddressPageChange, showTotal: (t) => `共 ${t} 笔`,
              }}
            />
          </Card>
        </div>
      ) : (
        <Card title={<span><SwapOutlined style={{ marginRight: 8 }} />链上交易</span>} size="small">
          {recentTxs.length > 0 ? (
            <Table columns={txColumns} dataSource={recentTxs} rowKey="tx_hash" size="small"
              pagination={{
                current: recentPage, total: recentTotal, pageSize: 15,
                onChange: (p) => fetchRecent(p),
                showTotal: (t) => `共 ${t} 笔交易`,
                showQuickJumper: true,
              }}
            />
          ) : (
            <Empty description="暂无交易数据，等待 RDS 开始处理交易后自动填充" />
          )}
        </Card>
      )}
    </div>
  );
};

export default TransactionExplorer;
