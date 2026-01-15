-- 事件日志表
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(66) NOT NULL,
    contract_address VARCHAR(42) NOT NULL,
    event_name VARCHAR(255),
    event_signature VARCHAR(66),
    topics JSONB,
    data TEXT,
    log_index INT,
    decoded_data JSONB,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_tx_hash ON events(tx_hash);
CREATE INDEX idx_events_contract ON events(contract_address);
CREATE INDEX idx_events_name ON events(event_name);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
