# 智能合约运行时风险监控系统

Smart Contract Runtime Risk Monitoring System (SCRRMS)

## 项目简介

本系统是一个基于微服务架构的智能合约运行时风险实时监控与告警平台，旨在通过动态监测、实时分析与自动预警，提升以太坊及兼容链上智能合约的安全防护能力。

## 系统架构

系统采用四层微服务架构：

```
用户层（React 前端）
    ↓ HTTP/WebSocket
API 接口层（API Gateway）
    ↓ RESTful API
业务服务层（RMS / RDS / 规则引擎）
    ↓ Kafka / PostgreSQL / Redis
数据存储层（PostgreSQL + Redis + Redpanda）
```

## 核心功能

- **运行时监控（RMS）**：通过 WebSocket 实时监听以太坊区块链交易和事件，提取调用栈和事件日志
- **风险检测（RDS）**：基于规则引擎检测重入攻击、闪电贷攻击、权限滥用、异常 Gas 消耗等安全风险
- **规则引擎**：基于 YAML 的灵活规则配置，支持条件组合（AND/OR）、风险评分因子、热加载
- **实时告警**：通过 WebSocket 实时推送风险事件到前端
- **可视化分析**：风险趋势图表、统计面板、事件列表、规则管理
- **系统状态监控**：服务健康检查、组件状态展示

## 技术栈

### 后端
- Go 1.24
- PostgreSQL 14 (关系型存储)
- Redis 7 (缓存 + Pub/Sub)
- Redpanda (Kafka 兼容消息队列)
- go-ethereum (以太坊客户端)
- gorilla/mux + gorilla/websocket (HTTP/WebSocket)

### 前端
- React 19 + TypeScript
- Ant Design 6 (UI 组件库)
- Recharts (数据可视化)
- Axios (HTTP 客户端)
- WebSocket (实时通信)

### 容器化
- Podman / Docker
- Docker Compose

## 项目结构

```
bcscan/
├── backend/
│   ├── cmd/
│   │   ├── api/          # API Gateway（HTTP + WebSocket）
│   │   ├── rms/          # Runtime Monitoring Service（区块链数据采集）
│   │   └── rds/          # Risk Detection Service（风险检测）
│   ├── internal/
│   │   ├── cache/        # Redis 客户端
│   │   ├── kafka/        # Kafka 生产者/消费者
│   │   ├── models/       # 数据模型
│   │   ├── repository/   # 数据访问层（Cache-Aside）
│   │   ├── ruleengine/   # 规则引擎（加载/求值/评分/执行）
│   │   │   └── hooks/    # Hook 系统（合约函数调用检测）
│   │   └── types/        # 共享类型定义
│   ├── migrations/       # 数据库迁移脚本
│   └── rules/
│       ├── builtin/      # 内置检测规则
│       └── custom/       # 自定义规则
├── frontend/             # React 前端应用
│   └── src/
│       ├── api/          # API 客户端 + WebSocket
│       └── pages/        # 页面组件
├── deployments/          # Docker Compose 配置
├── scripts/              # 开发脚本
└── docs/                 # 文档
```

## 快速开始

### 前置要求

- Go 1.21+
- Node.js 18+
- Podman 4.0+ (或 Docker)
- podman-compose (或 docker-compose)

### 启动开发环境

```bash
# 启动所有后端服务
./scripts/start-dev.sh

# 启动前端开发服务器
cd frontend && npm install && npm start

# 生成模拟交易（可选）
./scripts/start-event-generator.sh
```

### 服务访问地址

| 服务 | 地址 |
|------|------|
| 前端 | http://localhost:3000 |
| API Gateway | http://localhost:8080 |
| WebSocket | ws://localhost:8080/api/ws |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| Redpanda | localhost:9092 |
| Ganache | http://localhost:8545 |

### API 接口

#### 风险事件
- `GET /api/risks` - 分页获取风险事件列表（支持 severity/search/page/page_size）
- `GET /api/risks/{id}` - 获取单个风险事件详情

#### 统计分析
- `GET /api/stats` - 获取风险统计数据
- `GET /api/stats/trend?range=24h` - 获取风险趋势数据（支持 1h/6h/24h/7d）

#### 规则管理
- `GET /api/rules` - 获取所有检测规则
- `POST /api/rules/reload` - 热加载规则

#### 系统状态
- `GET /api/health` - 服务健康检查
- `GET /api/ws` - WebSocket 实时推送

## 内置检测规则

| 规则 | 严重程度 | 检测目标 |
|------|----------|----------|
| reentrancy-attack | Critical | 重入攻击模式（A→B→A 调用链） |
| flashloan-detection | Critical | 闪电贷攻击（高调用深度 + 高 Gas） |
| large-value-transfer | High | 大额 ETH 转账（> 1 ETH） |
| permission-abuse-detection | High | 权限滥用（多层代理调用） |
| abnormal-gas-consumption | Medium | 异常 Gas 消耗（> 1M Gas） |

## 规则热加载

```bash
# 修改规则文件后，触发热加载
curl -X POST http://localhost:8080/api/rules/reload

# 查看当前规则
curl http://localhost:8080/api/rules
```

## 开发文档

详细文档请查看 [docs](./docs) 目录。

## License

MIT
