# 智能合约运行时风险监控系统

## 项目简介

本系统是一个基于区块链的智能合约运行时风险实时监控与告警平台，旨在通过动态监测、实时分析与自动预警，提升以太坊及兼容链上智能合约的安全防护能力。

## 核心功能

- **运行时监控**：实时监听区块链交易和事件
- **风险检测**：检测重入攻击、闪电贷、权限滥用等安全风险
- **智能告警**：多渠道实时告警通知
- **可视化分析**：风险态势展示和报告生成
- **规则引擎**：基于YAML的灵活规则配置

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
- 用户管理服务: http://localhost:8001
- 运行时监控服务: http://localhost:8002
- 风险检测服务: http://localhost:8003
- 报告生成服务: http://localhost:8004
- 告警服务: http://localhost:8005

## 项目结构

```
bcscan/
├── backend/           # 后端服务
├── frontend/          # 前端应用
├── deployments/       # 部署配置
├── scripts/           # 脚本工具
├── docs/              # 文档
└── rules/             # 检测规则
```

## 开发文档

详细文档请查看 [docs](./docs) 目录。

## License

MIT
