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
}

export interface LoginResponse {
  token: string;
  expires_at: number;
  user: AuthUser;
}

// ==================== Token 管理 ====================

const TOKEN_KEY = 'bcscan_token';

export const getToken = (): string | null => localStorage.getItem(TOKEN_KEY);
export const setToken = (token: string) => localStorage.setItem(TOKEN_KEY, token);
export const removeToken = () => localStorage.removeItem(TOKEN_KEY);

// ==================== API Functions ====================

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
      // 如果不是在登录页，则跳转
      if (window.location.pathname !== '/login') {
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

// 认证 API
export const register = (data: { username: string; email: string; password: string }) => {
  return api.post<APIResponse<AuthUser>>('/auth/register', data);
};

export const login = (data: { username: string; password: string }) => {
  return api.post<APIResponse<LoginResponse>>('/auth/login', data);
};

export const getCurrentUser = () => {
  return api.get<APIResponse<AuthUser>>('/auth/me');
};

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

export const getStats = () => {
  return api.get<APIResponse<RiskStats>>('/stats');
};

export const getTrend = (range: string = '24h') => {
  return api.get<APIResponse<TrendPoint[]>>('/stats/trend', { params: { range } });
};

export const getRules = () => {
  return api.get<APIResponse<Rule[]>>('/rules');
};

export const reloadRules = () => {
  return api.post<APIResponse<{ message: string; count: number }>>('/rules/reload');
};

export const getHealth = () => {
  return api.get<HealthStatus>('/health');
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
