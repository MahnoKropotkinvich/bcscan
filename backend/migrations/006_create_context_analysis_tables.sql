-- 006: 地址行为画像 + 权限注册表
-- 支持跨交易上下文分析和权限提升检测

-- =============================================================================
-- 1. 地址行为画像表 —— 记录每个地址的累积行为统计
-- =============================================================================
CREATE TABLE IF NOT EXISTS address_profiles (
    address         VARCHAR(42)     PRIMARY KEY,
    -- 累计统计
    total_tx_count  BIGINT          NOT NULL DEFAULT 0,
    total_value_out NUMERIC(78,0)   NOT NULL DEFAULT 0,   -- 发出的总 wei
    total_value_in  NUMERIC(78,0)   NOT NULL DEFAULT 0,   -- 收到的总 wei
    total_gas_spent NUMERIC(78,0)   NOT NULL DEFAULT 0,
    -- 合约交互
    unique_contracts_called INT     NOT NULL DEFAULT 0,
    contract_deploy_count   INT     NOT NULL DEFAULT 0,
    -- 特权标记
    is_privileged   BOOLEAN         NOT NULL DEFAULT FALSE,
    privilege_roles TEXT[]          DEFAULT '{}',          -- 例如 ['owner','admin','operator']
    -- 风险标记
    risk_score      DECIMAL(5,2)    NOT NULL DEFAULT 0,
    risk_flags      TEXT[]          DEFAULT '{}',
    -- 时间
    first_seen_at   TIMESTAMP       NOT NULL DEFAULT NOW(),
    last_active_at  TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP       NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_address_profiles_privileged ON address_profiles (is_privileged) WHERE is_privileged = TRUE;
CREATE INDEX IF NOT EXISTS idx_address_profiles_risk_score ON address_profiles (risk_score DESC);
CREATE INDEX IF NOT EXISTS idx_address_profiles_last_active ON address_profiles (last_active_at DESC);

-- =============================================================================
-- 2. 权限注册表 —— 记录哪些合约/函数是特权操作
-- =============================================================================
CREATE TABLE IF NOT EXISTS privilege_registry (
    id              SERIAL          PRIMARY KEY,
    contract_address VARCHAR(42)    NOT NULL,
    -- 函数级权限
    function_selector VARCHAR(10)   NOT NULL DEFAULT '*',  -- '*' 表示整个合约
    function_name   VARCHAR(128)    NOT NULL DEFAULT '',
    -- 权限定义
    privilege_level VARCHAR(20)     NOT NULL DEFAULT 'high', -- critical/high/medium
    required_role   VARCHAR(64)     NOT NULL DEFAULT 'owner',
    authorized_addresses TEXT[]     DEFAULT '{}',            -- 有权调用的地址列表
    -- 描述
    description     TEXT            NOT NULL DEFAULT '',
    -- 元数据
    auto_detected   BOOLEAN         NOT NULL DEFAULT FALSE,  -- 是否自动检测到
    enabled         BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP       NOT NULL DEFAULT NOW(),
    UNIQUE(contract_address, function_selector)
);

CREATE INDEX IF NOT EXISTS idx_privilege_registry_contract ON privilege_registry (contract_address);
CREATE INDEX IF NOT EXISTS idx_privilege_registry_enabled ON privilege_registry (enabled) WHERE enabled = TRUE;

-- =============================================================================
-- 3. 地址交互记录 —— 滑动窗口用的原始数据
-- =============================================================================
CREATE TABLE IF NOT EXISTS address_interactions (
    id              BIGSERIAL       PRIMARY KEY,
    from_address    VARCHAR(42)     NOT NULL,
    to_address      VARCHAR(42)     NOT NULL,
    tx_hash         VARCHAR(66)     NOT NULL,
    block_number    BIGINT          NOT NULL,
    value           NUMERIC(78,0)   NOT NULL DEFAULT 0,
    gas_used        BIGINT          NOT NULL DEFAULT 0,
    function_selector VARCHAR(10)   NOT NULL DEFAULT '',
    call_depth      INT             NOT NULL DEFAULT 0,
    has_delegatecall BOOLEAN        NOT NULL DEFAULT FALSE,
    status          SMALLINT        NOT NULL DEFAULT 1,    -- 0=failed, 1=success
    timestamp       TIMESTAMP       NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_address_interactions_from ON address_interactions (from_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_address_interactions_to ON address_interactions (to_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_address_interactions_pair ON address_interactions (from_address, to_address, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_address_interactions_ts ON address_interactions (timestamp DESC);

-- 分区裁剪提示：如果数据量大可以按月分区，此处暂不分区

-- =============================================================================
-- 4. 预置一些已知特权函数签名（通用 Solidity 模式）
-- =============================================================================
INSERT INTO privilege_registry (contract_address, function_selector, function_name, privilege_level, required_role, description, auto_detected)
VALUES
    ('*', '0xf2fde38b', 'transferOwnership(address)', 'critical', 'owner', '转移合约所有权', true),
    ('*', '0x715018a6', 'renounceOwnership()', 'critical', 'owner', '放弃合约所有权', true),
    ('*', '0x8da5cb5b', 'owner()', 'low', '*', '查询所有者（只读）', true),
    ('*', '0xff9b54ab', 'setAdmin(address)', 'critical', 'owner', '设置管理员', true),
    ('*', '0x3659cfe6', 'upgradeTo(address)', 'critical', 'admin', '升级合约实现（代理模式）', true),
    ('*', '0x4f1ef286', 'upgradeToAndCall(address,bytes)', 'critical', 'admin', '升级合约实现并调用', true),
    ('*', '0x3f4ba83a', 'unpause()', 'high', 'owner', '取消暂停合约', true),
    ('*', '0x8456cb59', 'pause()', 'high', 'owner', '暂停合约', true),
    ('*', '0x40c10f19', 'mint(address,uint256)', 'high', 'minter', '铸造代币', true),
    ('*', '0x42966c68', 'burn(uint256)', 'medium', '*', '销毁代币', true),
    ('*', '0x2f2ff15d', 'grantRole(bytes32,address)', 'critical', 'admin', '授予角色权限', true),
    ('*', '0xd547741f', 'revokeRole(bytes32,address)', 'critical', 'admin', '撤销角色权限', true),
    ('*', '0x00f714ce', 'withdraw(uint256,address)', 'high', 'owner', '提取资金到指定地址', true),
    ('*', '0x51cff8d9', 'withdraw(address)', 'high', 'owner', '提取全部资金', true),
    ('*', '0xff9b54ab', 'selfDestruct(address)', 'critical', 'owner', '自毁合约', true)
ON CONFLICT (contract_address, function_selector) DO NOTHING;
