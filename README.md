# 智能合约运行时风险监控系统

## 项目简介

本系统是一个基于区块链的智能合约运行时风险实时监控与告警平台，旨在通过动态监测、实时分析与自动预警，提升以太坊及兼容链上智能合约的安全防护能力。

## 核心功能

- **运行时监控**：实时监听区块链交易和事件
- **风险检测**：检测重入攻击、闪电贷、权限滥用等安全风险
- **智能告警**：多渠道实时告警通知
- **可视化分析**：风险态势展示和报告生成
- **规则引擎**：基于YAML的灵活规则配置
- **规则热加载**：支持在线更新规则，无需重启服务

## 技术栈

### 后端
- Go 1.21+
- PostgreSQL 14+
- Redis 7+
- Redpanda (Kafka兼容)

### 前端
- React 18+
- TypeScript
- Ant Design

### 容器化
- Podman
- Podman Compose

## 快速开始

### 前置要求

- Go 1.21+
- Node.js 18+
- Podman 4.0+
- podman-compose

### 安装 podman-compose

```bash
pip3 install podman-compose
```

### 启动开发环境

```bash
# 启动所有服务
./scripts/start-dev.sh

# 查看服务状态
podman-compose -f deployments/podman-compose.yml ps

# 查看日志
./scripts/logs.sh <service-name>

# 停止服务
./scripts/stop-dev.sh
```

### 服务访问地址

- 前端: http://localhost:3000
- API Gateway: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- Redpanda: localhost:9092
- Ganache: http://localhost:8545

### API 接口

#### 风险事件
- `GET /api/risks` - 获取风险事件列表
- `GET /api/risks/{id}` - 获取单个风险事件
- `GET /api/stats` - 获取统计数据

#### 规则管理
- `GET /api/rules` - 获取所有规则
- `POST /api/rules/reload` - 重新加载规则并触发热更新

## 项目结构

```
bcscan/
├── backend/
│   ├── cmd/
│   │   ├── api/          # API Gateway
│   │   ├── rms/          # Runtime Monitoring Service
│   │   └── rds/          # Risk Detection Service
│   ├── internal/
│   │   ├── cache/        # Redis 客户端
│   │   ├── kafka/        # Kafka 客户端
│   │   ├── models/       # 数据模型
│   │   └── ruleengine/   # 规则引擎
│   ├── migrations/       # 数据库迁移
│   └── rules/            # 内置规则
├── frontend/             # React 前端
├── deployments/          # Docker Compose 配置
└── docs/                 # 文档
```

## 规则热加载

系统支持在线更新检测规则，无需重启服务：

### 工作原理
1. 规则首次加载时缓存到 Redis
2. RDS 服务订阅 Redis Pub/Sub 频道
3. 调用 API 重新加载规则时，自动发布更新通知
4. 所有 RDS 实例收到通知后自动重新加载规则

### 使用方法

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
