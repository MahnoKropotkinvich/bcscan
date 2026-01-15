-- 风险事件表
CREATE TABLE IF NOT EXISTS risk_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    contract_address VARCHAR(42),
    tx_hash VARCHAR(66),
    description TEXT,
    evidence JSONB,
    score DECIMAL(5,2),
    status VARCHAR(20) DEFAULT 'detected',
    detected_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_risk_events_type ON risk_events(event_type);
CREATE INDEX idx_risk_events_severity ON risk_events(severity);
CREATE INDEX idx_risk_events_contract ON risk_events(contract_address);
CREATE INDEX idx_risk_events_tx ON risk_events(tx_hash);
CREATE INDEX idx_risk_events_detected_at ON risk_events(detected_at DESC);

-- 检测规则表
CREATE TABLE IF NOT EXISTS detection_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    rule_type VARCHAR(50),
    expression TEXT,
    severity VARCHAR(20),
    enabled BOOLEAN DEFAULT true,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rules_name ON detection_rules(name);
CREATE INDEX idx_rules_enabled ON detection_rules(enabled);
