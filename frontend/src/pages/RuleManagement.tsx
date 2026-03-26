import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Button, Space, message, Spin, Descriptions, Modal, Badge } from 'antd';
import { ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { getRules, reloadRules, Rule } from '../api';

const RuleManagement: React.FC = () => {
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(false);
  const [reloading, setReloading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedRule, setSelectedRule] = useState<Rule | null>(null);

  const fetchRules = async () => {
    setLoading(true);
    try {
      const res = await getRules();
      if (res.data.success) {
        setRules(res.data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch rules:', err);
      message.error('获取规则列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleReload = async () => {
    setReloading(true);
    try {
      const res = await reloadRules();
      if (res.data.success) {
        message.success(`规则热加载成功，共加载 ${(res.data.data as any).count} 条规则`);
        fetchRules();
      }
    } catch (err) {
      console.error('Failed to reload rules:', err);
      message.error('规则热加载失败');
    } finally {
      setReloading(false);
    }
  };

  const showDetail = (rule: Rule) => {
    setSelectedRule(rule);
    setDetailVisible(true);
  };

  const severityColors: Record<string, string> = {
    critical: 'red',
    high: 'orange',
    medium: 'blue',
    low: 'green',
  };

  const columns = [
    {
      title: '规则名称',
      dataIndex: ['metadata', 'name'],
      key: 'name',
      render: (name: string) => <strong>{name}</strong>,
    },
    {
      title: '版本',
      dataIndex: ['metadata', 'version'],
      key: 'version',
      width: 80,
    },
    {
      title: '严重程度',
      dataIndex: ['config', 'severity'],
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severityColors[severity] || 'default'}>{severity.toUpperCase()}</Tag>
      ),
    },
    {
      title: '优先级',
      dataIndex: ['config', 'priority'],
      key: 'priority',
      width: 80,
    },
    {
      title: '状态',
      dataIndex: ['metadata', 'enabled'],
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) =>
        enabled ? (
          <Badge status="success" text="启用" />
        ) : (
          <Badge status="default" text="禁用" />
        ),
    },
    {
      title: '描述',
      dataIndex: ['metadata', 'description'],
      key: 'description',
      ellipsis: true,
    },
    {
      title: '标签',
      dataIndex: ['metadata', 'tags'],
      key: 'tags',
      width: 200,
      render: (tags: string[]) =>
        tags?.map((tag) => (
          <Tag key={tag} style={{ marginBottom: 2 }}>
            {tag}
          </Tag>
        )),
    },
    {
      title: '触发条件数',
      key: 'conditions',
      width: 100,
      render: (_: any, record: Rule) => record.triggers?.conditions?.length || 0,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: any, record: Rule) => (
        <Button
          type="link"
          icon={<InfoCircleOutlined />}
          onClick={() => showDetail(record)}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>规则管理</h2>
        <Space>
          <span style={{ color: '#999' }}>共 {rules.length} 条规则</span>
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            loading={reloading}
            onClick={handleReload}
          >
            热加载规则
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={rules}
          rowKey={(record) => record.metadata.name}
          loading={loading}
          pagination={false}
          size="small"
        />
      </Card>

      <Modal
        title={`规则详情 - ${selectedRule?.metadata.name}`}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={700}
      >
        {selectedRule && (
          <div>
            <Descriptions bordered size="small" column={2} style={{ marginBottom: 16 }}>
              <Descriptions.Item label="规则名称">{selectedRule.metadata.name}</Descriptions.Item>
              <Descriptions.Item label="版本">{selectedRule.metadata.version}</Descriptions.Item>
              <Descriptions.Item label="作者">{selectedRule.metadata.author}</Descriptions.Item>
              <Descriptions.Item label="严重程度">
                <Tag color={severityColors[selectedRule.config.severity]}>
                  {selectedRule.config.severity.toUpperCase()}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="优先级">{selectedRule.config.priority}</Descriptions.Item>
              <Descriptions.Item label="状态">
                {selectedRule.metadata.enabled ? (
                  <Tag icon={<CheckCircleOutlined />} color="success">启用</Tag>
                ) : (
                  <Tag icon={<CloseCircleOutlined />} color="default">禁用</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>{selectedRule.metadata.description}</Descriptions.Item>
              <Descriptions.Item label="Hook">{selectedRule.config.hooks?.join(', ')}</Descriptions.Item>
              <Descriptions.Item label="基础分数">{selectedRule.scoring.base_score}</Descriptions.Item>
            </Descriptions>

            <Card title="触发条件" size="small" style={{ marginBottom: 16 }}>
              <p><strong>逻辑运算符：</strong>{selectedRule.triggers.operator}</p>
              <Table
                size="small"
                pagination={false}
                dataSource={selectedRule.triggers.conditions}
                rowKey={(_, i) => String(i)}
                columns={[
                  { title: '类型', dataIndex: 'type', key: 'type' },
                  { title: '运算符', dataIndex: 'operator', key: 'operator' },
                  { title: '阈值', dataIndex: 'value', key: 'value', render: (v: any) => String(v) },
                  { title: '描述', dataIndex: 'description', key: 'description' },
                ]}
              />
            </Card>

            <Card title="评分因子" size="small" style={{ marginBottom: 16 }}>
              <Table
                size="small"
                pagination={false}
                dataSource={selectedRule.scoring.factors}
                rowKey={(_, i) => String(i)}
                columns={[
                  { title: '条件', dataIndex: 'condition', key: 'condition' },
                  {
                    title: '分数变化',
                    dataIndex: 'score',
                    key: 'score',
                    render: (s: number) => (
                      <span style={{ color: s > 0 ? '#cf1322' : '#52c41a' }}>
                        {s > 0 ? '+' : ''}{s}
                      </span>
                    ),
                  },
                  { title: '描述', dataIndex: 'description', key: 'description' },
                ]}
              />
            </Card>

            {selectedRule.config.throttle?.enabled && (
              <Card title="限流配置" size="small">
                <Descriptions size="small" column={2}>
                  <Descriptions.Item label="最大告警数">{selectedRule.config.throttle.max_alerts}</Descriptions.Item>
                  <Descriptions.Item label="时间窗口">{selectedRule.config.throttle.time_window}</Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default RuleManagement;
