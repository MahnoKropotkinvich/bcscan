import React, { useEffect, useState, useCallback } from 'react';
import { Card, Row, Col, Statistic, Select, Table, Tag, Button, Dropdown, message, Spin } from 'antd';
import type { MenuProps } from 'antd';
import {
  BarChartOutlined, DownloadOutlined, FileTextOutlined,
  AlertOutlined, SafetyCertificateOutlined, RiseOutlined,
  FileOutlined,
} from '@ant-design/icons';
import {
  PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, Legend, ResponsiveContainer, AreaChart, Area,
} from 'recharts';
import {
  getReportSummary, getReportByType, getReportByContract,
  getReportTimeline, exportReport,
  ReportSummary, ReportByType, ReportByContract, ReportTimeline,
} from '../api';

const ReportCenter: React.FC = () => {
  const [timeRange, setTimeRange] = useState('7d');
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState<ReportSummary | null>(null);
  const [byType, setByType] = useState<ReportByType[]>([]);
  const [byContract, setByContract] = useState<ReportByContract[]>([]);
  const [timeline, setTimeline] = useState<ReportTimeline[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [sumRes, typeRes, contractRes, timelineRes] = await Promise.all([
        getReportSummary(timeRange),
        getReportByType(timeRange),
        getReportByContract(timeRange, 10),
        getReportTimeline(timeRange),
      ]);

      if (sumRes.data.success) setSummary(sumRes.data.data);
      if (typeRes.data.success) setByType(typeRes.data.data || []);
      if (contractRes.data.success) setByContract(contractRes.data.data || []);
      if (timelineRes.data.success) setTimeline(timelineRes.data.data || []);
    } catch {
      message.error('获取报告数据失败');
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleExportJSON = async () => {
    try {
      const res = await exportReport(timeRange);
      const blob = new Blob([res.data], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `risk-report-${timeRange}-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
      message.success('JSON 报告导出成功');
    } catch {
      message.error('报告导出失败');
    }
  };

  // 将数组对象转换为 CSV 字符串
  const toCSV = (rows: Record<string, any>[], headers: { key: string; label: string }[]): string => {
    const escape = (v: any) => {
      const s = String(v ?? '');
      // 含逗号、换行、双引号时用双引号包裹，内部双引号转义
      return s.includes(',') || s.includes('\n') || s.includes('"')
        ? `"${s.replace(/"/g, '""')}"`
        : s;
    };
    const headerRow = headers.map(h => h.label).join(',');
    const dataRows = rows.map(row => headers.map(h => escape(row[h.key])).join(','));
    return [headerRow, ...dataRows].join('\r\n');
  };

  const handleExportCSV = () => {
    try {
      const dateStr = new Date().toISOString().slice(0, 10);
      const sections: string[] = [];

      // ── 概要 ──
      sections.push('# 报告概要');
      sections.push(toCSV(
        [{
          time_range: timeRange,
          total_events: summary?.total_events ?? 0,
          avg_score: summary?.avg_score ?? '',
          total_alerts: summary?.total_alerts ?? 0,
          resolved_alerts: summary?.resolved_alerts ?? 0,
          alert_resolve_rate: summary?.alert_resolve_rate ?? '',
          generated_at: summary?.generated_at ?? '',
        }],
        [
          { key: 'time_range',          label: '时间范围' },
          { key: 'total_events',        label: '总事件数' },
          { key: 'avg_score',           label: '平均分数' },
          { key: 'total_alerts',        label: '告警总数' },
          { key: 'resolved_alerts',     label: '已解决告警' },
          { key: 'alert_resolve_rate',  label: '处理率' },
          { key: 'generated_at',        label: '生成时间' },
        ]
      ));

      // ── 按事件类型 ──
      sections.push('\r\n# 按事件类型统计');
      sections.push(toCSV(byType, [
        { key: 'event_type', label: '事件类型' },
        { key: 'severity',   label: '严重程度' },
        { key: 'count',      label: '数量' },
        { key: 'avg_score',  label: '平均分数' },
      ]));

      // ── 高危合约 Top 10 ──
      sections.push('\r\n# 高危合约 Top 10');
      sections.push(toCSV(byContract, [
        { key: 'contract_address', label: '合约地址' },
        { key: 'event_count',      label: '事件数' },
        { key: 'avg_score',        label: '平均分数' },
        { key: 'max_severity',     label: '最高等级' },
      ]));

      // ── 时间线 ──
      sections.push('\r\n# 风险事件时间线');
      sections.push(toCSV(timeline, [
        { key: 'time',     label: '时间' },
        { key: 'critical', label: '严重' },
        { key: 'high',     label: '高危' },
        { key: 'medium',   label: '中危' },
        { key: 'low',      label: '低危' },
        { key: 'total',    label: '合计' },
      ]));

      const csvContent = '\uFEFF' + sections.join('\r\n'); // BOM 头，Excel 正确识别中文
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `risk-report-${timeRange}-${dateStr}.csv`;
      a.click();
      URL.revokeObjectURL(url);
      message.success('CSV 报告导出成功');
    } catch {
      message.error('CSV 导出失败');
    }
  };

  const exportMenuItems: MenuProps['items'] = [
    {
      key: 'json',
      icon: <FileTextOutlined />,
      label: '导出 JSON',
      onClick: handleExportJSON,
    },
    {
      key: 'csv',
      icon: <FileOutlined />,
      label: '导出 CSV',
      onClick: handleExportCSV,
    },
  ];

  const severityColors: Record<string, string> = {
    critical: '#cf1322',
    high: '#fa8c16',
    medium: '#1890ff',
    low: '#52c41a',
  };

  // 饼图数据
  const pieData = summary?.by_severity?.map((s) => ({
    name: s.severity.toUpperCase(),
    value: s.count,
    fill: severityColors[s.severity] || '#999',
  })) || [];

  const contractColumns = [
    {
      title: '排名', key: 'rank', width: 50,
      render: (_: any, __: any, index: number) => <strong>{index + 1}</strong>,
    },
    {
      title: '合约地址', dataIndex: 'contract_address', key: 'contract_address',
      ellipsis: true,
      render: (addr: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }} title={addr}>{addr}</span>
      ),
    },
    {
      title: '事件数', dataIndex: 'event_count', key: 'event_count', width: 80,
      render: (c: number) => <strong style={{ color: '#cf1322' }}>{c}</strong>,
    },
    {
      title: '平均分数', dataIndex: 'avg_score', key: 'avg_score', width: 90,
    },
    {
      title: '最高等级', dataIndex: 'max_severity', key: 'max_severity', width: 90,
      render: (s: string) => <Tag color={severityColors[s]}>{s.toUpperCase()}</Tag>,
    },
  ];

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" tip="生成报告中..." />
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>
          <BarChartOutlined style={{ marginRight: 8 }} />
          报告中心
        </h2>
        <div style={{ display: 'flex', gap: 12 }}>
          <Select value={timeRange} onChange={setTimeRange} style={{ width: 120 }}>
            <Select.Option value="24h">最近24小时</Select.Option>
            <Select.Option value="7d">最近7天</Select.Option>
            <Select.Option value="30d">最近30天</Select.Option>
            <Select.Option value="90d">最近90天</Select.Option>
          </Select>
          <Dropdown menu={{ items: exportMenuItems }} placement="bottomRight">
            <Button icon={<DownloadOutlined />} type="primary">
              导出报告 ▾
            </Button>
          </Dropdown>
        </div>
      </div>

      {/* 概要统计 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={4}>
          <Card size="small">
            <Statistic title="总事件数" value={summary?.total_events || 0} prefix={<AlertOutlined />} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title="平均分数" value={summary?.avg_score || '0'} prefix={<RiseOutlined />} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title="告警总数" value={summary?.total_alerts || 0} prefix={<SafetyCertificateOutlined />} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title="已解决" value={summary?.resolved_alerts || 0}
              valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title="处理率" value={summary?.alert_resolve_rate || 'N/A'} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title="生成时间" value={summary?.generated_at ? new Date(summary.generated_at).toLocaleTimeString('zh-CN') : '-'}
              valueStyle={{ fontSize: 16 }} />
          </Card>
        </Col>
      </Row>

      {/* 图表行：饼图 + 趋势 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card title="风险等级分布" size="small">
            {pieData.length > 0 ? (
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%"
                    outerRadius={80} label={({ name, value }) => `${name}: ${value}`}>
                    {pieData.map((entry, i) => (
                      <Cell key={i} fill={entry.fill} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div style={{ textAlign: 'center', padding: 60, color: '#999' }}>暂无数据</div>
            )}
          </Card>
        </Col>
        <Col span={16}>
          <Card title="风险事件时间线" size="small">
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={timeline}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Area type="monotone" dataKey="critical" stroke="#cf1322" fill="#cf1322" fillOpacity={0.3} name="严重" stackId="1" />
                <Area type="monotone" dataKey="high" stroke="#fa8c16" fill="#fa8c16" fillOpacity={0.3} name="高危" stackId="1" />
                <Area type="monotone" dataKey="medium" stroke="#1890ff" fill="#1890ff" fillOpacity={0.3} name="中危" stackId="1" />
                <Area type="monotone" dataKey="low" stroke="#52c41a" fill="#52c41a" fillOpacity={0.3} name="低危" stackId="1" />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      {/* 按事件类型统计柱状图 */}
      <Card title="按事件类型统计" size="small" style={{ marginBottom: 24 }}>
        {byType.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={byType}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="event_type" angle={-15} textAnchor="end" height={60} />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey="count" name="事件数">
                {byType.map((entry, i) => (
                  <Cell key={i} fill={severityColors[entry.severity] || '#1890ff'} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div style={{ textAlign: 'center', padding: 60, color: '#999' }}>暂无数据</div>
        )}
      </Card>

      {/* 合约风险 Top N */}
      <Card title={<span><FileTextOutlined style={{ marginRight: 8 }} />高危合约 Top 10</span>} size="small">
        <Table columns={contractColumns} dataSource={byContract}
          rowKey="contract_address" pagination={false} size="small" />
      </Card>
    </div>
  );
};

export default ReportCenter;
