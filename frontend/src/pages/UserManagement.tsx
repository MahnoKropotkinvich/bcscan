import React, { useEffect, useState, useCallback } from 'react';
import { Card, Table, Tag, Button, Select, Input, message, Spin, Modal, Row, Col, Statistic, Space } from 'antd';
import { TeamOutlined, UserOutlined, SafetyCertificateOutlined, StopOutlined, CheckCircleOutlined } from '@ant-design/icons';
import {
  getUsers, getRolesInfo, updateUserRole, updateUserStatus,
  AuthUser, RoleInfo, ROLE_LABELS, ROLE_COLORS,
} from '../api';

const { Search } = Input;

const UserManagement: React.FC = () => {
  const [users, setUsers] = useState<AuthUser[]>([]);
  const [roles, setRoles] = useState<RoleInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [roleFilter, setRoleFilter] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 15;

  // 修改角色 modal
  const [roleModalVisible, setRoleModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<AuthUser | null>(null);
  const [newRole, setNewRole] = useState('');

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const params: any = { page, page_size: pageSize };
      if (roleFilter && roleFilter !== 'all') params.role = roleFilter;
      if (search) params.search = search;

      const res = await getUsers(params);
      if (res.data.success) {
        setUsers(res.data.data.items || []);
        setTotal(res.data.data.total || 0);
      }
    } catch (err) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, roleFilter, search]);

  const fetchRoles = useCallback(async () => {
    try {
      const res = await getRolesInfo();
      if (res.data.success) {
        setRoles(res.data.data || []);
      }
    } catch { /* ignore */ }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  useEffect(() => {
    fetchRoles();
  }, [fetchRoles]);

  const handleSearch = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handleRoleFilter = (value: string) => {
    setRoleFilter(value);
    setPage(1);
  };

  const openRoleModal = (user: AuthUser) => {
    setEditingUser(user);
    setNewRole(user.role);
    setRoleModalVisible(true);
  };

  const handleUpdateRole = async () => {
    if (!editingUser) return;
    try {
      await updateUserRole(editingUser.id, newRole);
      message.success('角色更新成功');
      setRoleModalVisible(false);
      fetchUsers();
      fetchRoles();
    } catch (err: any) {
      message.error(err.response?.data?.error || '角色更新失败');
    }
  };

  const handleToggleStatus = async (user: AuthUser) => {
    const newStatus = user.status === 'active' ? 'disabled' : 'active';
    const actionText = newStatus === 'active' ? '启用' : '禁用';

    Modal.confirm({
      title: `确认${actionText}用户`,
      content: `确定要${actionText}用户 "${user.username}" 吗？`,
      onOk: async () => {
        try {
          await updateUserStatus(user.id, newStatus);
          message.success(`用户已${actionText}`);
          fetchUsers();
        } catch (err: any) {
          message.error(err.response?.data?.error || `${actionText}失败`);
        }
      },
    });
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (name: string) => (
        <span>
          <UserOutlined style={{ marginRight: 4 }} />
          <strong>{name}</strong>
        </span>
      ),
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 120,
      render: (role: string) => (
        <Tag color={ROLE_COLORS[role] || 'default'}>
          {ROLE_LABELS[role] || role}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) =>
        status === 'active' ? (
          <Tag icon={<CheckCircleOutlined />} color="success">正常</Tag>
        ) : (
          <Tag icon={<StopOutlined />} color="error">禁用</Tag>
        ),
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 170,
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
    {
      title: '最后登录',
      dataIndex: 'last_login_at',
      key: 'last_login_at',
      width: 170,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: any, record: AuthUser) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => openRoleModal(record)}>
            改角色
          </Button>
          <Button
            type="link"
            size="small"
            danger={record.status === 'active'}
            onClick={() => handleToggleStatus(record)}
          >
            {record.status === 'active' ? '禁用' : '启用'}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>
        <TeamOutlined style={{ marginRight: 8 }} />
        用户管理
      </h2>

      {/* 角色统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {roles.map((r) => (
          <Col span={Math.floor(24 / Math.max(roles.length, 1))} key={r.role}>
            <Card size="small">
              <Statistic
                title={r.label}
                value={r.count}
                prefix={<SafetyCertificateOutlined />}
                valueStyle={{ color: ROLE_COLORS[r.role] === 'default' ? '#666' : undefined }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* 搜索和筛选 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          <Search
            placeholder="搜索用户名或邮箱"
            style={{ width: 300 }}
            onSearch={handleSearch}
            allowClear
            enterButton
          />
          <Select
            defaultValue="all"
            style={{ width: 150 }}
            onChange={handleRoleFilter}
          >
            <Select.Option value="all">全部角色</Select.Option>
            {Object.entries(ROLE_LABELS).map(([k, v]) => (
              <Select.Option key={k} value={k}>{v}</Select.Option>
            ))}
          </Select>
          <span style={{ color: '#999' }}>共 {total} 个用户</span>
        </div>
      </Card>

      {/* 用户列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            total,
            pageSize,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p) => setPage(p),
          }}
          size="small"
        />
      </Card>

      {/* 修改角色 Modal */}
      <Modal
        title={`修改角色 - ${editingUser?.username}`}
        open={roleModalVisible}
        onOk={handleUpdateRole}
        onCancel={() => setRoleModalVisible(false)}
      >
        <div style={{ padding: '16px 0' }}>
          <p>当前角色：<Tag color={ROLE_COLORS[editingUser?.role || '']}>{ROLE_LABELS[editingUser?.role || ''] || editingUser?.role}</Tag></p>
          <p style={{ marginTop: 16 }}>新角色：</p>
          <Select value={newRole} onChange={setNewRole} style={{ width: '100%' }}>
            {Object.entries(ROLE_LABELS).map(([k, v]) => (
              <Select.Option key={k} value={k}>{v}</Select.Option>
            ))}
          </Select>
        </div>
      </Modal>
    </div>
  );
};

export default UserManagement;
