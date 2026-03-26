-- ============================================================
-- 008: 交易浏览器 + 函数签名库 (Batch 6)
-- ============================================================

-- 给 transactions 表添加完整调用链和事件日志字段
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS function_selector VARCHAR(10);
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS call_stack JSONB;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS events_data JSONB;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS gas_limit BIGINT;

CREATE INDEX IF NOT EXISTS idx_transactions_func ON transactions(function_selector);

-- 函数签名库（内置 4byte directory）
CREATE TABLE IF NOT EXISTS function_signatures (
    id BIGSERIAL PRIMARY KEY,
    selector VARCHAR(10) NOT NULL,       -- 0xaabbccdd
    signature VARCHAR(500) NOT NULL,     -- transfer(address,uint256)
    name VARCHAR(200) NOT NULL,          -- transfer
    category VARCHAR(50),                -- token, defi, governance, access_control, proxy
    description TEXT,
    is_privileged BOOLEAN DEFAULT false, -- 是否是特权函数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_func_sig_selector_sig ON function_signatures(selector, signature);
CREATE INDEX IF NOT EXISTS idx_func_sig_selector ON function_signatures(selector);
CREATE INDEX IF NOT EXISTS idx_func_sig_category ON function_signatures(category);

-- 预置常见函数签名（ERC20/ERC721 + DeFi + 治理 + 代理）
INSERT INTO function_signatures (selector, signature, name, category, description, is_privileged) VALUES
    -- ERC20
    ('0xa9059cbb', 'transfer(address,uint256)', 'transfer', 'token', 'ERC20代币转账', false),
    ('0x23b872dd', 'transferFrom(address,address,uint256)', 'transferFrom', 'token', 'ERC20授权转账', false),
    ('0x095ea7b3', 'approve(address,uint256)', 'approve', 'token', 'ERC20授权额度', false),
    ('0x70a08231', 'balanceOf(address)', 'balanceOf', 'token', '查询余额', false),
    ('0x18160ddd', 'totalSupply()', 'totalSupply', 'token', '查询总供应量', false),
    ('0xdd62ed3e', 'allowance(address,address)', 'allowance', 'token', '查询授权额度', false),
    -- ERC721
    ('0x42842e0e', 'safeTransferFrom(address,address,uint256)', 'safeTransferFrom', 'token', 'ERC721安全转账', false),
    ('0x6352211e', 'ownerOf(uint256)', 'ownerOf', 'token', '查询NFT所有者', false),
    ('0xb88d4fde', 'safeTransferFrom(address,address,uint256,bytes)', 'safeTransferFrom', 'token', 'ERC721安全转账(带数据)', false),
    -- 所有权管理
    ('0xf2fde38b', 'transferOwnership(address)', 'transferOwnership', 'access_control', '转移合约所有权', true),
    ('0x715018a6', 'renounceOwnership()', 'renounceOwnership', 'access_control', '放弃合约所有权', true),
    ('0x8da5cb5b', 'owner()', 'owner', 'access_control', '查询合约所有者', false),
    -- 访问控制(OpenZeppelin)
    ('0x2f2ff15d', 'grantRole(bytes32,address)', 'grantRole', 'access_control', '授予角色', true),
    ('0xd547741f', 'revokeRole(bytes32,address)', 'revokeRole', 'access_control', '撤销角色', true),
    ('0x91d14854', 'hasRole(bytes32,address)', 'hasRole', 'access_control', '检查角色', false),
    ('0x36568abe', 'renounceRole(bytes32,address)', 'renounceRole', 'access_control', '放弃角色', true),
    ('0x248a9ca3', 'getRoleAdmin(bytes32)', 'getRoleAdmin', 'access_control', '获取角色管理员', false),
    -- 代理模式
    ('0x3659cfe6', 'upgradeTo(address)', 'upgradeTo', 'proxy', '升级合约实现', true),
    ('0x4f1ef286', 'upgradeToAndCall(address,bytes)', 'upgradeToAndCall', 'proxy', '升级并调用', true),
    ('0x5c60da1b', 'implementation()', 'implementation', 'proxy', '查询实现地址', false),
    ('0xf851a440', 'admin()', 'admin', 'proxy', '查询代理管理员', false),
    ('0x8f283970', 'changeAdmin(address)', 'changeAdmin', 'proxy', '更换代理管理员', true),
    -- DeFi 常见
    ('0x38ed1739', 'swapExactTokensForTokens(uint256,uint256,address[],address,uint256)', 'swapExactTokensForTokens', 'defi', 'Uniswap V2精确输入兑换', false),
    ('0x8803dbee', 'swapTokensForExactTokens(uint256,uint256,address[],address,uint256)', 'swapTokensForExactTokens', 'defi', 'Uniswap V2精确输出兑换', false),
    ('0x7ff36ab5', 'swapExactETHForTokens(uint256,address[],address,uint256)', 'swapExactETHForTokens', 'defi', 'ETH兑换代币', false),
    ('0xe8e33700', 'addLiquidity(address,address,uint256,uint256,uint256,uint256,address,uint256)', 'addLiquidity', 'defi', '添加流动性', false),
    ('0xbaa2abde', 'removeLiquidity(address,address,uint256,uint256,uint256,address,uint256)', 'removeLiquidity', 'defi', '移除流动性', false),
    ('0xb6b55f25', 'deposit(uint256)', 'deposit', 'defi', '存款', false),
    ('0x2e1a7d4d', 'withdraw(uint256)', 'withdraw', 'defi', '取款', false),
    ('0xa0712d68', 'mint(uint256)', 'mint', 'token', '铸造代币', true),
    ('0x40c10f19', 'mint(address,uint256)', 'mint', 'token', '铸造代币到指定地址', true),
    ('0x42966c68', 'burn(uint256)', 'burn', 'token', '销毁代币', false),
    -- 治理
    ('0xda95691a', 'propose(address[],uint256[],string[],bytes[],string)', 'propose', 'governance', '提交治理提案', false),
    ('0x56781388', 'castVote(uint256,uint8)', 'castVote', 'governance', '投票', false),
    ('0x2656209d', 'queue(uint256)', 'queue', 'governance', '提案入队', false),
    ('0xfe0d94c1', 'execute(uint256)', 'execute', 'governance', '执行提案', true),
    -- Pausable
    ('0x8456cb59', 'pause()', 'pause', 'access_control', '暂停合约', true),
    ('0x3f4ba83a', 'unpause()', 'unpause', 'access_control', '恢复合约', true),
    -- 通用
    ('0x3ccfd60b', 'withdraw()', 'withdraw', 'defi', '提取资金', false),
    ('0xd0e30db0', 'deposit()', 'deposit', 'defi', '存入资金', false)
ON CONFLICT (selector, signature) DO NOTHING;
