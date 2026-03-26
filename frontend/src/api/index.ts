import axios from 'axios';

const API_URL = process.env.REACT_APP_API_URL || '/api';

export const api = axios.create({
  baseURL: API_URL,
  timeout: 15000,
});

// ==================== Types ====================

export interface RiskEvent {
  id: number;
  event_type: string;
  severity: string;
  contract_address: string;
  tx_hash: string;
  description: string;
  score: number;
  detected_at: string;
}

export interface PagedResult<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

export interface APIResponse<T> {
  success: boolean;
  data: T;
  error?: string;
  total?: number;
}

export interface RiskStats {
  total: number;
  by_severity: Record<string, number>;
  last_24h: number;
  last_1h: number;
}

export interface TrendPoint {
  time: string;
  critical: number;
  high: number;
  medium: number;
  low: number;
}

export interface Rule {
  metadata: {
    name: string;
    version: string;
    author: string;
    description: string;
    tags: string[];
    enabled: boolean;
  };
  config: {
    severity: string;
    priority: number;
    hooks: string[];
    throttle?: {
      enabled: boolean;
      max_alerts: number;
      time_window: string;
    };
  };
  triggers: {
    operator: string;
    conditions: {
      type: string;
      operator: string;
      value: any;
      description: string;
    }[];
  };
  scoring: {
    base_score: number;
    factors: {
      condition: string;
      score: number;
      description: string;
    }[];
  };
  actions: {
    type: string;
    severity?: string;
    title?: string;
    message?: string;
  }[];
}

export interface HealthStatus {
  status: string;
  timestamp: string;
  services: Record<string, string>;
}

export interface AuthUser {
  id: number;
  username: string;
  email: string;
  role: string;
  status: string;
  created_at: string;
  last_login_at?: string;
}

export interface LoginResponse {
  token: string;
  expires_at: number;
  user: AuthUser;
}

// ==================== 告警相关类型 ====================

export interface Alert {
  id: number;
  risk_event_id: number;
  title: string;
  message: string;
  severity: string;
  status: string; // pending, acknowledged, resolved, ignored
  assigned_to?: number;
  acknowledged_at?: string;
  acknowledged_by?: number;
  resolved_at?: string;
  resolved_by?: number;
  notes?: string;
  created_at: string;
  tx_hash?: string;
  contract_address?: string;
  score?: number;
}

export interface AlertHistory {
  id: number;
  alert_id: number;
  user_id?: number;
  username: string;
  action: string;
  old_status: string;
  new_status: string;
  note: string;
  created_at: string;
}

export interface AlertStats {
  by_status: Record<string, number>;
  pending_count: number;
  total: number;
}

export interface AlertDetail {
  alert: Alert;
  history: AlertHistory[];
}

// ==================== 报告相关类型 ====================

export interface ReportSummary {
  total_events: number;
  by_severity: { severity: string; count: number }[];
  avg_score: string;
  alert_resolve_rate: string;
  total_alerts: number;
  resolved_alerts: number;
  time_range: string;
  generated_at: string;
}

export interface ReportByType {
  event_type: string;
  severity: string;
  count: number;
  avg_score: string;
}

export interface ReportByContract {
  contract_address: string;
  event_count: number;
  avg_score: string;
  max_severity: string;
}

export interface ReportTimeline {
  time: string;
  critical: number;
  high: number;
  medium: number;
  low: number;
  total: number;
}

// ==================== 审计日志类型 ====================

export interface AuditLogEntry {
  id: number;
  user_id?: number;
  username: string;
  action: string;
  resource: string;
  details: string;
  ip_address: string;
  created_at: string;
}

// ==================== 角色相关类型 ====================

export interface RoleInfo {
  role: string;
  label: string;
  count: number;
}

// 角色标签映射
export const ROLE_LABELS: Record<string, string> = {
  admin: '系统管理员',
  analyst: '安全分析师',
  developer: 'DApp开发者',
  operator: '运维人员',
  user: '普通用户',
};

// 角色颜色
export const ROLE_COLORS: Record<string, string> = {
  admin: 'red',
  analyst: 'blue',
  developer: 'green',
  operator: 'orange',
  user: 'default',
};

// ==================== Token 管理 ====================

const TOKEN_KEY = 'bcscan_token';

export const getToken = (): string | null => localStorage.getItem(TOKEN_KEY);
export const setToken = (token: string) => localStorage.setItem(TOKEN_KEY, token);
export const removeToken = () => localStorage.removeItem(TOKEN_KEY);

// ==================== 拦截器 ====================

// 请求拦截器：自动附加 JWT
api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截器：401 时清除 token
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      removeToken();
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

// ==================== 认证 API ====================

export const register = (data: { username: string; email: string; password: string; role?: string }) => {
  return api.post<APIResponse<AuthUser>>('/auth/register', data);
};

export const login = (data: { username: string; password: string }) => {
  return api.post<APIResponse<LoginResponse>>('/auth/login', data);
};

export const getCurrentUser = () => {
  return api.get<APIResponse<AuthUser>>('/auth/me');
};

// ==================== 风险事件 API ====================

export const getRiskEvents = (params?: {
  severity?: string;
  search?: string;
  page?: number;
  page_size?: number;
}) => {
  return api.get<PagedResult<RiskEvent>>('/risks', { params });
};

export const getRiskEvent = (id: number) => {
  return api.get<APIResponse<RiskEvent>>(`/risks/${id}`);
};

// ==================== 统计 API ====================

export const getStats = () => {
  return api.get<APIResponse<RiskStats>>('/stats');
};

export const getTrend = (range: string = '24h') => {
  return api.get<APIResponse<TrendPoint[]>>('/stats/trend', { params: { range } });
};

// ==================== 规则 API ====================

export const getRules = () => {
  return api.get<APIResponse<Rule[]>>('/rules');
};

export const reloadRules = () => {
  return api.post<APIResponse<{ message: string; count: number }>>('/rules/reload');
};

// ==================== 系统状态 API ====================

export const getHealth = () => {
  return api.get<HealthStatus>('/health');
};

// ==================== 告警 API ====================

export const getAlerts = (params?: {
  status?: string;
  severity?: string;
  search?: string;
  page?: number;
  page_size?: number;
}) => {
  return api.get<PagedResult<Alert>>('/alerts', { params });
};

export const getAlertStats = () => {
  return api.get<APIResponse<AlertStats>>('/alerts/stats');
};

export const getAlertDetail = (id: number) => {
  return api.get<APIResponse<AlertDetail>>(`/alerts/${id}`);
};

export const acknowledgeAlert = (id: number) => {
  return api.post<APIResponse<{ message: string }>>(`/alerts/${id}/acknowledge`);
};

export const resolveAlert = (id: number, note?: string) => {
  return api.post<APIResponse<{ message: string }>>(`/alerts/${id}/resolve`, { note });
};

export const ignoreAlert = (id: number, note?: string) => {
  return api.post<APIResponse<{ message: string }>>(`/alerts/${id}/ignore`, { note });
};

export const addAlertNote = (id: number, note: string) => {
  return api.post<APIResponse<{ message: string }>>(`/alerts/${id}/note`, { note });
};

// ==================== 报告 API ====================

export const getReportSummary = (range: string = '7d') => {
  return api.get<APIResponse<ReportSummary>>('/reports/summary', { params: { range } });
};

export const getReportByType = (range: string = '7d') => {
  return api.get<APIResponse<ReportByType[]>>('/reports/by-type', { params: { range } });
};

export const getReportByContract = (range: string = '7d', limit: number = 10) => {
  return api.get<APIResponse<ReportByContract[]>>('/reports/by-contract', { params: { range, limit } });
};

export const getReportTimeline = (range: string = '7d') => {
  return api.get<APIResponse<ReportTimeline[]>>('/reports/timeline', { params: { range } });
};

export const exportReport = (range: string = '7d') => {
  return api.get('/reports/export', { params: { range }, responseType: 'blob' });
};

// ==================== 用户管理 API ====================

export const getUsers = (params?: {
  role?: string;
  search?: string;
  page?: number;
  page_size?: number;
}) => {
  return api.get<APIResponse<PagedResult<AuthUser>>>('/users', { params });
};

export const getRolesInfo = () => {
  return api.get<APIResponse<RoleInfo[]>>('/users/roles');
};

export const updateUserRole = (id: number, role: string) => {
  return api.put<APIResponse<{ message: string }>>(`/users/${id}/role`, { role });
};

export const updateUserStatus = (id: number, status: string) => {
  return api.put<APIResponse<{ message: string }>>(`/users/${id}/status`, { status });
};

// ==================== 审计日志 API ====================

export const getAuditLogs = (params?: {
  action?: string;
  search?: string;
  page?: number;
  page_size?: number;
}) => {
  return api.get<APIResponse<PagedResult<AuditLogEntry>>>('/audit-logs', { params });
};

export const getAuditActions = () => {
  return api.get<APIResponse<string[]>>('/audit-logs/actions');
};

// ==================== 交易浏览器类型 ====================

export interface ExplorerTransaction {
  tx_hash: string;
  block_number: number;
  from_address: string;
  to_address: string;
  value: string;
  gas_price: number;
  gas_used: number;
  gas_limit?: number;
  input_data?: string;
  function_selector?: string;
  function_name?: string;
  function_desc?: string;
  status: number;
  timestamp: string;
  call_stack?: CallFrame[];
  events_data?: EventLogData[];
  risk_count: number;
}

export interface CallFrame {
  type: string;
  from: string;
  to: string;
  value: string;
  gas: number;
  gas_used: number;
  input: string;
  output: string;
  error: string;
  depth: number;
  function: string;
  function_name?: string;
  function_desc?: string;
}

export interface EventLogData {
  address: string;
  topics: string[];
  data: string;
}

export interface TxBrief {
  tx_hash: string;
  block_number: number;
  from_address: string;
  to_address: string;
  value: string;
  gas_used: number;
  function_selector?: string;
  function_name?: string;
  status: number;
  timestamp: string;
}

export interface AddressSummary {
  address: string;
  tx_count: number;
  sent_count: number;
  received_count: number;
  risk_count: number;
  top_functions?: { selector: string; name: string; count: number }[];
}

export interface FunctionSignature {
  selector: string;
  signature: string;
  name: string;
  category: string;
  description: string;
  is_privileged: boolean;
}

// ==================== 交易浏览器 API ====================

export const getTransactionByHash = (hash: string) => {
  return api.get<APIResponse<ExplorerTransaction>>(`/explorer/tx/${hash}`);
};

export const getTransactionsByAddress = (address: string, params?: { page?: number; page_size?: number }) => {
  return api.get<PagedResult<TxBrief>>(`/explorer/address/${address}/txs`, { params });
};

export const getAddressSummary = (address: string) => {
  return api.get<APIResponse<AddressSummary>>(`/explorer/address/${address}/summary`);
};

export const getRisksByTxHash = (hash: string) => {
  return api.get<APIResponse<RiskEvent[]>>(`/explorer/tx/${hash}/risks`);
};

export const decodeFunctionSelector = (selector: string) => {
  return api.get<APIResponse<FunctionSignature[]>>(`/explorer/decode/${selector}`);
};

export const getRecentTransactions = (limit: number = 20) => {
  return api.get<APIResponse<TxBrief[]>>('/explorer/recent', { params: { limit } });
};

// ==================== WebSocket ====================

const WS_URL = process.env.REACT_APP_WS_URL ||
  `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/ws`;

export type WSMessageHandler = (event: RiskEvent) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private handlers: WSMessageHandler[] = [];
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldReconnect = true;

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    try {
      this.ws = new WebSocket(WS_URL);

      this.ws.onopen = () => {
        console.log('[WebSocket] Connected');
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as RiskEvent;
          this.handlers.forEach((handler) => handler(data));
        } catch (e) {
          console.warn('[WebSocket] Failed to parse message:', e);
        }
      };

      this.ws.onclose = () => {
        console.log('[WebSocket] Disconnected');
        if (this.shouldReconnect) {
          this.reconnectTimer = setTimeout(() => this.connect(), 3000);
        }
      };

      this.ws.onerror = (error) => {
        console.warn('[WebSocket] Error:', error);
      };
    } catch (e) {
      console.warn('[WebSocket] Connection failed:', e);
      if (this.shouldReconnect) {
        this.reconnectTimer = setTimeout(() => this.connect(), 3000);
      }
    }
  }

  disconnect() {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }
    this.ws?.close();
  }

  onMessage(handler: WSMessageHandler) {
    this.handlers.push(handler);
    return () => {
      this.handlers = this.handlers.filter((h) => h !== handler);
    };
  }
}

// 全局单例
export const wsClient = new WebSocketClient();
