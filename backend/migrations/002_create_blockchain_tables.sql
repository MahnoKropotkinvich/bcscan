-- 监控合约表
CREATE TABLE IF NOT EXISTS monitored_contracts (
    id BIGSERIAL PRIMARY KEY,
    address VARCHAR(42) UNIQUE NOT NULL,
    name VARCHAR(255),
    abi JSONB,
    chain_id INT NOT NULL DEFAULT 1,
    status VARCHAR(20) DEFAULT 'active',
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_contracts_address ON monitored_contracts(address);
CREATE INDEX idx_contracts_chain_id ON monitored_contracts(chain_id);
CREATE INDEX idx_contracts_status ON monitored_contracts(status);

-- 区块表
CREATE TABLE IF NOT EXISTS blocks (
    id BIGSERIAL PRIMARY KEY,
    block_number BIGINT UNIQUE NOT NULL,
    block_hash VARCHAR(66) UNIQUE NOT NULL,
    parent_hash VARCHAR(66),
    timestamp TIMESTAMP NOT NULL,
    miner VARCHAR(42),
    gas_used BIGINT,
    gas_limit BIGINT,
    transaction_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_blocks_number ON blocks(block_number DESC);
CREATE INDEX idx_blocks_hash ON blocks(block_hash);
CREATE INDEX idx_blocks_timestamp ON blocks(timestamp DESC);

-- 交易表（分区表，按月分区）
CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL,
    tx_hash VARCHAR(66) UNIQUE NOT NULL,
    block_number BIGINT NOT NULL,
    from_address VARCHAR(42) NOT NULL,
    to_address VARCHAR(42),
    value NUMERIC(78, 0),
    gas_price BIGINT,
    gas_used BIGINT,
    input_data TEXT,
    status SMALLINT DEFAULT 1,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

CREATE INDEX idx_transactions_hash ON transactions(tx_hash);
CREATE INDEX idx_transactions_block ON transactions(block_number);
CREATE INDEX idx_transactions_from ON transactions(from_address);
CREATE INDEX idx_transactions_to ON transactions(to_address);
CREATE INDEX idx_transactions_timestamp ON transactions(timestamp DESC);

-- 创建初始分区（当前月份）
CREATE TABLE IF NOT EXISTS transactions_2024_01 PARTITION OF transactions
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
