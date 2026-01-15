# RDS (Risk Detection Service)

智能合约运行时风险监控服务

## 功能

- 实时监控区块链交易
- 基于规则引擎检测风险模式
- 支持多种风险评分因子
- 自动执行告警和记录动作

## 架构

```
Kafka (Redpanda) -> RDS Service -> PostgreSQL
                         |
                    Rule Engine
                    - Evaluator
                    - Hooks
                    - Scorer
                    - Executor
```

## 配置

通过环境变量配置：

```bash
DATABASE_URL=postgres://user:pass@localhost:5432/bcscan?sslmode=disable
KAFKA_BROKER=localhost:9092
KAFKA_TOPIC=blockchain.transactions
RULES_PATH=./rules/builtin
```

## 运行

```bash
# 编译
go build -o rds

# 运行
./rds
```

## 规则格式

规则文件使用 YAML 格式，示例：

```yaml
metadata:
  name: "rule-name"
  enabled: true

config:
  severity: "critical"
  hooks:
    - "post_transaction"

triggers:
  operator: "AND"
  conditions:
    - type: "call_depth"
      operator: ">"
      value: 3

scoring:
  base_score: 85
  factors:
    - condition: "call_depth > 5"
      score: 10

actions:
  - type: "alert"
    severity: "critical"
  - type: "log_risk_event"
```

## 消息格式

Kafka 消息格式（JSON）：

```json
{
  "tx_hash": "0x...",
  "block_number": 12345,
  "from_address": "0x...",
  "to_address": "0x...",
  "value": "1000000000000000000",
  "gas_price": 20000000000,
  "gas_used": 21000,
  "input_data": "0x",
  "status": 1,
  "timestamp": "2024-01-15T10:00:00Z"
}
```
