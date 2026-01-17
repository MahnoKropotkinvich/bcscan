import React, { useEffect, useState, useRef } from 'react';
import { Card, Row, Col, Statistic, Table, Radio } from 'antd';
import { AlertOutlined, CheckCircleOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface RiskStats {
  total: number;
  critical: number;
  high: number;
  medium: number;
}

interface ChartDataPoint {
  timestamp: number;
  time: string;
  critical: number;
  high: number;
  medium: number;
}

interface EventRecord {
  timestamp: number;
  severity: 'critical' | 'high' | 'medium';
}

const RULES = ['重入攻击检测', '大额转账', '异常Gas消耗', '权限滥用', '闪电贷攻击'];

const getRandomSeverity = () => {
  const rand = Math.random();
  if (rand < 0.1) return 'critical';  // 10%
  if (rand < 0.4) return 'high';      // 30%
  return 'medium';                     // 60%
};

const generateTxHash = () => '0x' + Math.random().toString(16).substr(2, 8) + '...' + Math.random().toString(16).substr(2, 3);
const getTimeAgo = (minutes: number) => minutes < 60 ? `${minutes}分钟前` : `${Math.floor(minutes / 60)}小时前`;

const aggregateEvents = (events: EventRecord[], timeRange: string): ChartDataPoint[] => {
  const now = Date.now();
  const ranges: any = {
    '1m': { duration: 60000, buckets: 12, bucketSize: 5000 },
    '30m': { duration: 1800000, buckets: 30, bucketSize: 60000 },
    '1h': { duration: 3600000, buckets: 12, bucketSize: 300000 },
    '24h': { duration: 86400000, buckets: 24, bucketSize: 3600000 },
  };
  const config = ranges[timeRange];
  
  const result: ChartDataPoint[] = [];
  for (let i = 0; i < config.buckets; i++) {
    const bucketEnd = now - (config.buckets - 1 - i) * config.bucketSize;
    const bucketStart = bucketEnd - config.bucketSize;
    
    const bucketEvents = events.filter(e => e.timestamp >= bucketStart && e.timestamp < bucketEnd);
    const d = new Date(bucketEnd);
    
    result.push({
      timestamp: bucketEnd,
      time: timeRange === '24h' 
        ? `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:00`
        : d.toTimeString().substr(0, timeRange === '1m' ? 8 : 5),
      critical: bucketEvents.filter(e => e.severity === 'critical').length,
      high: bucketEvents.filter(e => e.severity === 'high').length,
      medium: bucketEvents.filter(e => e.severity === 'medium').length,
    });
  }
  return result;
};

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<RiskStats>({ total: 0, critical: 0, high: 0, medium: 0 });
  const [recentEvents, setRecentEvents] = useState<any[]>([]);
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [timeRange, setTimeRange] = useState<string>('1h');
  const eventHistoryRef = useRef<EventRecord[]>([]);
  const initializedRef = useRef(false);

  useEffect(() => {
    if (initializedRef.current) return;
    initializedRef.current = true;

    // 初始化数据
    const initialTotal = Math.floor(Math.random() * 50) + 100;
    setStats({
      total: initialTotal,
      critical: Math.floor(initialTotal * 0.1),
      high: Math.floor(initialTotal * 0.3),
      medium: Math.floor(initialTotal * 0.6),
    });

    // 初始化历史事件（过去24小时）
    const now = Date.now();
    for (let i = 0; i < 50; i++) {
      eventHistoryRef.current.push({
        timestamp: now - Math.random() * 86400000,
        severity: getRandomSeverity(),
      });
    }
    eventHistoryRef.current.sort((a, b) => a.timestamp - b.timestamp);

    // 初始化最近事件（基于历史数据）
    const recentHistory = eventHistoryRef.current.slice(-5);
    const initialEvents = recentHistory.map((e, i) => ({
      id: Date.now() + i,
      rule: RULES[Math.floor(Math.random() * RULES.length)],
      severity: e.severity,
      tx_hash: generateTxHash(),
      time: getTimeAgo(Math.floor(Math.random() * 120)),
    }));
    setRecentEvents(initialEvents);
    
    setChartData(aggregateEvents(eventHistoryRef.current, timeRange));
  }, []);

  useEffect(() => {
    // 每5秒更新数据
    const interval = setInterval(() => {
      console.log('[Dashboard] 更新数据...');
      
      setStats(prev => {
        const newTotal = prev.total + 1;
        return {
          total: newTotal,
          critical: prev.critical + (Math.random() > 0.8 ? 1 : 0),
          high: prev.high + (Math.random() > 0.6 ? 1 : 0),
          medium: prev.medium + (Math.random() > 0.4 ? 1 : 0),
        };
      });

      // 每次都添加新事件
      const severity = getRandomSeverity();
      const newEvent = {
        id: Date.now(),
        rule: RULES[Math.floor(Math.random() * RULES.length)],
        severity,
        tx_hash: generateTxHash(),
        time: '刚刚',
      };
      
      setRecentEvents(prev => [newEvent, ...prev.slice(0, 4)]);
      console.log('[Dashboard] 新事件:', newEvent.rule);

      // 添加到历史记录
      eventHistoryRef.current.push({
        timestamp: Date.now(),
        severity,
      });
      
      // 只保留24小时内的数据
      const cutoff = Date.now() - 86400000;
      eventHistoryRef.current = eventHistoryRef.current.filter(e => e.timestamp > cutoff);
      
      // 更新图表
      setChartData(aggregateEvents(eventHistoryRef.current, timeRange));
    }, 5000);

    return () => clearInterval(interval);
  }, [timeRange]);

  useEffect(() => {
    // 只在切换时间尺度时重新聚合
    setChartData(aggregateEvents(eventHistoryRef.current, timeRange));
  }, [timeRange]);

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

      <Card 
        title="风险趋势" 
        style={{ marginBottom: 24 }}
        extra={
          <Radio.Group value={timeRange} onChange={(e) => setTimeRange(e.target.value)}>
            <Radio.Button value="1m">1分钟</Radio.Button>
            <Radio.Button value="30m">30分钟</Radio.Button>
            <Radio.Button value="1h">1小时</Radio.Button>
            <Radio.Button value="24h">24小时</Radio.Button>
          </Radio.Group>
        }
      >
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={chartData}>
            <defs>
              <linearGradient id="colorCritical" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#cf1322" stopOpacity={0.8}/>
                <stop offset="95%" stopColor="#cf1322" stopOpacity={0.1}/>
              </linearGradient>
              <linearGradient id="colorHigh" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#fa8c16" stopOpacity={0.8}/>
                <stop offset="95%" stopColor="#fa8c16" stopOpacity={0.1}/>
              </linearGradient>
              <linearGradient id="colorMedium" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#1890ff" stopOpacity={0.8}/>
                <stop offset="95%" stopColor="#1890ff" stopOpacity={0.1}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="time" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Area type="monotone" dataKey="critical" stroke="#cf1322" fillOpacity={1} fill="url(#colorCritical)" name="严重" isAnimationActive={false} />
            <Area type="monotone" dataKey="high" stroke="#fa8c16" fillOpacity={1} fill="url(#colorHigh)" name="高危" isAnimationActive={false} />
            <Area type="monotone" dataKey="medium" stroke="#1890ff" fillOpacity={1} fill="url(#colorMedium)" name="中危" isAnimationActive={false} />
          </AreaChart>
        </ResponsiveContainer>
      </Card>

      <Card title="最近风险事件">
        <Table columns={columns} dataSource={recentEvents} rowKey="id" pagination={false} />
      </Card>
    </div>
  );
};

export default Dashboard;
