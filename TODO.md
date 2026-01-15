# BCScan 项目任务清单

## 已完成 ✅

### 基础设施
- [x] PostgreSQL 数据库配置
- [x] Redis 缓存配置
- [x] Redpanda 消息队列配置
- [x] Ganache 测试网络配置
- [x] Docker Compose 网络配置 (bcscan)

### 数据库
- [x] 数据库迁移脚本
  - [x] 用户表
  - [x] 区块链数据表
  - [x] 事件表
  - [x] 风险事件表
  - [x] 告警表

### 后端服务
- [x] RDS (Risk Detection Service) - 风险检测服务
  - [x] 规则引擎核心
    - [x] 表达式求值器 (evaluator.go)
    - [x] Hook 系统 (hooks/)
    - [x] 风险评分器 (scorer.go)
    - [x] 动作执行器 (executor.go)
  - [x] 规则加载器
  - [x] Kafka 消费者集成
  - [x] 数据库集成
  - [x] 容器化部署
  - [x] 内置规则：重入攻击检测

### Kafka 集成
- [x] Producer 实现
- [x] Consumer 实现
- [x] Redpanda 连接测试

## 进行中 🚧

### RMS (Runtime Monitoring Service) - 运行时监控服务
- [ ] 连接 Ganache 节点
- [ ] 监听区块和交易
- [ ] 解析交易数据
- [ ] 发送数据到 Kafka
- [ ] 容器化部署

## 待完成 📋

### 后端服务

#### RMS (Runtime Monitoring Service) - 优先级：高
- [ ] 实时监控智能合约执行
- [ ] 提取交易调用栈
- [ ] 分析 Gas 使用情况
- [ ] 检测异常行为模式
- [ ] 性能优化

#### UMS (User Management Service) - 优先级：中
- [ ] 用户注册/登录
- [ ] JWT 认证
- [ ] 权限管理 (RBAC)
- [ ] API 密钥管理
- [ ] 用户配置管理
- [ ] 容器化部署

#### AS (Alert Service) - 优先级：中
- [ ] 告警规则配置
- [ ] 多渠道通知
  - [ ] 邮件通知
  - [ ] Webhook
  - [ ] Slack 集成
- [ ] 告警历史记录
- [ ] 告警统计分析
- [ ] 容器化部署

#### RGS (Report Generation Service) - 优先级：低
- [ ] 风险报告生成
- [ ] PDF 导出
- [ ] 数据可视化图表
- [ ] 定期报告调度
- [ ] 报告模板管理
- [ ] 容器化部署

### 规则引擎增强
- [ ] 支持更多条件类型
  - [ ] repeated_call
  - [ ] state_change
  - [ ] balance_change
- [ ] 自定义规则 DSL
- [ ] 规则测试框架
- [ ] 规则性能优化

### 前端 (React.js) - 优先级：高
- [ ] 项目初始化
  - [ ] Create React App / Vite
  - [ ] TypeScript 配置
  - [ ] UI 框架选择 (Ant Design / Material-UI)
- [ ] 页面开发
  - [ ] 登录/注册页面
  - [ ] 仪表板 (Dashboard)
  - [ ] 实时监控页面
  - [ ] 风险事件列表
  - [ ] 规则管理页面
  - [ ] 告警配置页面
  - [ ] 报告查看页面
  - [ ] 用户设置页面
- [ ] 数据可视化
  - [ ] 实时交易图表
  - [ ] 风险趋势图
  - [ ] 统计面板
- [ ] WebSocket 实时更新
- [ ] 容器化部署

### API Gateway
- [ ] 统一 API 入口
- [ ] 路由配置
- [ ] 认证中间件
- [ ] 限流控制
- [ ] API 文档 (Swagger)
- [ ] 容器化部署

### 测试
- [ ] 单元测试
- [ ] 集成测试
- [ ] E2E 测试
- [ ] 性能测试
- [ ] 安全测试

### 文档
- [ ] API 文档
- [ ] 部署文档
- [ ] 用户手册
- [ ] 开发者指南
- [ ] 架构设计文档

### DevOps
- [ ] CI/CD 流水线
- [ ] 监控告警 (Prometheus + Grafana)
- [ ] 日志聚合 (ELK)
- [ ] 备份策略
- [ ] 灾难恢复方案

## 技术栈

### 后端
- Go 1.24
- PostgreSQL 14
- Redis 7
- Redpanda (Kafka API)
- go-ethereum

### 前端
- React.js 18
- TypeScript
- Ant Design / Material-UI
- Chart.js / ECharts
- WebSocket

### 基础设施
- Docker / Podman
- Docker Compose
- Nginx (反向代理)

## 优先级说明
- **高**: 核心功能，必须完成
- **中**: 重要功能，尽快完成
- **低**: 增强功能，可延后

## 下一步行动
1. 实现 RMS 服务 (本周)
2. 开发前端基础框架 (本周)
3. 实现 UMS 和 AS (下周)
4. 完善前端页面 (下周)
5. 实现 RGS (后续)
