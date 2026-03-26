package profile

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// PrivilegeEntry 权限注册条目
type PrivilegeEntry struct {
	ID                  int      `json:"id"`
	ContractAddress     string   `json:"contract_address"`  // '*' = 通配
	FunctionSelector    string   `json:"function_selector"` // '*' = 整个合约
	FunctionName        string   `json:"function_name"`
	PrivilegeLevel      string   `json:"privilege_level"`      // critical/high/medium/low
	RequiredRole        string   `json:"required_role"`        // owner/admin/minter/*
	AuthorizedAddresses []string `json:"authorized_addresses"` // 有权调用的地址
	Description         string   `json:"description"`
	Enabled             bool     `json:"enabled"`
}

// PrivilegeCheckResult 权限检查结果
type PrivilegeCheckResult struct {
	IsPrivilegedCall bool            `json:"is_privileged_call"`
	CallerAuthorized bool            `json:"caller_authorized"`
	PrivilegeLevel   string          `json:"privilege_level"`
	RequiredRole     string          `json:"required_role"`
	MatchedEntry     *PrivilegeEntry `json:"matched_entry,omitempty"`
	// 调用链分析
	HasDelegatecall       bool     `json:"has_delegatecall"`
	EffectiveCaller       string   `json:"effective_caller"`       // 最终实际调用者
	CallerIsContract      bool     `json:"caller_is_contract"`     // 调用者是合约而非 EOA
	IntermediaryContracts []string `json:"intermediary_contracts"` // 中间代理合约
}

// PrivilegeRegistry 权限注册表
type PrivilegeRegistry struct {
	db     *sql.DB
	logger *zap.Logger

	// 内存缓存（启动时全量加载，热更新）
	entries []PrivilegeEntry
	// 按函数选择器索引: funcSel -> []*PrivilegeEntry
	bySelector map[string][]*PrivilegeEntry
	// 按合约地址索引: contract -> []*PrivilegeEntry
	byContract map[string][]*PrivilegeEntry
	mu         sync.RWMutex
}

// NewPrivilegeRegistry 创建权限注册表
func NewPrivilegeRegistry(db *sql.DB, logger *zap.Logger) *PrivilegeRegistry {
	pr := &PrivilegeRegistry{
		db:         db,
		logger:     logger,
		bySelector: make(map[string][]*PrivilegeEntry),
		byContract: make(map[string][]*PrivilegeEntry),
	}
	return pr
}

// Load 从数据库加载所有权限条目
func (pr *PrivilegeRegistry) Load(ctx context.Context) error {
	rows, err := pr.db.QueryContext(ctx, `
		SELECT id, contract_address, function_selector, function_name,
		       privilege_level, required_role, authorized_addresses,
		       description, enabled
		FROM privilege_registry
		WHERE enabled = TRUE
		ORDER BY privilege_level, contract_address
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var entries []PrivilegeEntry
	for rows.Next() {
		var e PrivilegeEntry
		if err := rows.Scan(
			&e.ID, &e.ContractAddress, &e.FunctionSelector, &e.FunctionName,
			&e.PrivilegeLevel, &e.RequiredRole, pq.Array(&e.AuthorizedAddresses),
			&e.Description, &e.Enabled,
		); err != nil {
			return err
		}
		entries = append(entries, e)
	}

	// 重建索引
	bySelector := make(map[string][]*PrivilegeEntry)
	byContract := make(map[string][]*PrivilegeEntry)
	for i := range entries {
		e := &entries[i]
		sel := strings.ToLower(e.FunctionSelector)
		bySelector[sel] = append(bySelector[sel], e)
		contract := strings.ToLower(e.ContractAddress)
		byContract[contract] = append(byContract[contract], e)
	}

	pr.mu.Lock()
	pr.entries = entries
	pr.bySelector = bySelector
	pr.byContract = byContract
	pr.mu.Unlock()

	pr.logger.Info("Privilege registry loaded", zap.Int("entries", len(entries)))
	return nil
}

// CheckPrivilege 检查一次调用是否涉及特权操作
// 返回权限检查结果
func (pr *PrivilegeRegistry) CheckPrivilege(
	caller string,
	contractAddress string,
	functionSelector string,
	callStack []CallFrameInfo,
) *PrivilegeCheckResult {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := &PrivilegeCheckResult{
		EffectiveCaller: strings.ToLower(caller),
	}

	// 分析调用链
	pr.analyzeCallChain(callStack, result)

	// 查找匹配的权限条目
	entry := pr.findMatchingEntry(contractAddress, functionSelector)
	if entry == nil {
		return result
	}

	result.IsPrivilegedCall = true
	result.PrivilegeLevel = entry.PrivilegeLevel
	result.RequiredRole = entry.RequiredRole
	result.MatchedEntry = entry

	// 检查调用者是否有权限
	result.CallerAuthorized = pr.isAuthorized(result.EffectiveCaller, entry)

	return result
}

// CallFrameInfo 调用帧信息（精简版，从 types.CallFrame 转换而来）
type CallFrameInfo struct {
	Type  string // CALL, DELEGATECALL, STATICCALL, CREATE
	From  string
	To    string
	Depth int
	Input string // 用于提取 function selector
}

// findMatchingEntry 查找匹配的权限条目
func (pr *PrivilegeRegistry) findMatchingEntry(contractAddress, functionSelector string) *PrivilegeEntry {
	contract := strings.ToLower(contractAddress)
	funcSel := strings.ToLower(functionSelector)

	// 1. 精确匹配: 特定合约 + 特定函数
	if entries, ok := pr.byContract[contract]; ok {
		for _, e := range entries {
			if strings.ToLower(e.FunctionSelector) == funcSel {
				return e
			}
		}
		// 匹配合约级通配
		for _, e := range entries {
			if e.FunctionSelector == "*" {
				return e
			}
		}
	}

	// 2. 通配合约匹配: * + 特定函数
	if entries, ok := pr.byContract["*"]; ok {
		for _, e := range entries {
			if strings.ToLower(e.FunctionSelector) == funcSel {
				return e
			}
		}
	}

	return nil
}

// analyzeCallChain 分析调用链，检测 delegatecall 和中间代理
func (pr *PrivilegeRegistry) analyzeCallChain(callStack []CallFrameInfo, result *PrivilegeCheckResult) {
	intermediaries := make(map[string]bool)

	for _, frame := range callStack {
		if strings.EqualFold(frame.Type, "DELEGATECALL") {
			result.HasDelegatecall = true
		}
		// 中间合约（非最终目标的 to 地址都是中间人）
		if frame.To != "" && frame.Depth > 0 {
			intermediaries[strings.ToLower(frame.To)] = true
		}
	}

	for addr := range intermediaries {
		result.IntermediaryContracts = append(result.IntermediaryContracts, addr)
	}

	// 如果调用链中有 delegatecall，有效调用者可能不是 tx.from
	// 在 delegatecall 链中，msg.sender 会保持为外层调用者
	// 但如果 EOA -> ContractA.delegatecall(ContractB.privilegedFunc())
	// 那么在 ContractB 的视角中，msg.sender = ContractA，而非 EOA
	// 对于检测目的，我们关注 tx.from（最外层 EOA）是否有权限
	if len(callStack) > 0 {
		// 检查 tx.from 是否是 EOA（没有在 callStack 中作为 to 出现）
		if _, isContract := intermediaries[result.EffectiveCaller]; isContract {
			result.CallerIsContract = true
		}
	}
}

// isAuthorized 检查调用者是否被授权
func (pr *PrivilegeRegistry) isAuthorized(caller string, entry *PrivilegeEntry) bool {
	// 如果 RequiredRole 是 "*"，任何人都可以
	if entry.RequiredRole == "*" {
		return true
	}

	// 检查是否在授权地址列表中
	for _, addr := range entry.AuthorizedAddresses {
		if strings.EqualFold(addr, caller) {
			return true
		}
	}

	return false
}

// RegisterPrivilege 注册新的权限条目
func (pr *PrivilegeRegistry) RegisterPrivilege(ctx context.Context, entry *PrivilegeEntry) error {
	_, err := pr.db.ExecContext(ctx, `
		INSERT INTO privilege_registry (contract_address, function_selector, function_name,
		                                 privilege_level, required_role, authorized_addresses,
		                                 description, auto_detected, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE)
		ON CONFLICT (contract_address, function_selector)
		DO UPDATE SET
			privilege_level = EXCLUDED.privilege_level,
			authorized_addresses = EXCLUDED.authorized_addresses,
			updated_at = NOW()
	`, entry.ContractAddress, entry.FunctionSelector, entry.FunctionName,
		entry.PrivilegeLevel, entry.RequiredRole, pq.Array(entry.AuthorizedAddresses),
		entry.Description, entry.Enabled)
	if err != nil {
		return err
	}

	// 重新加载
	return pr.Load(ctx)
}

// AutoDetectOwnership 自动检测 OwnershipTransferred 事件并注册 owner
// 当看到 OwnershipTransferred(oldOwner, newOwner) 事件时，更新权限表
func (pr *PrivilegeRegistry) AutoDetectOwnership(ctx context.Context, contractAddress string, newOwner string, tracker *Tracker) error {
	addr := strings.ToLower(contractAddress)
	owner := strings.ToLower(newOwner)

	// 标记新 owner 为特权地址
	if tracker != nil {
		_ = tracker.SetPrivileged(ctx, owner, []string{"owner"})
	}

	// 更新该合约所有权限条目的授权地址（把 newOwner 加进去）
	_, err := pr.db.ExecContext(ctx, `
		UPDATE privilege_registry
		SET authorized_addresses = array_append(
			array_remove(authorized_addresses, $2), $2
		), updated_at = NOW()
		WHERE (contract_address = $1 OR contract_address = '*')
		  AND required_role = 'owner'
		  AND enabled = TRUE
	`, addr, owner)
	if err != nil {
		return err
	}

	pr.logger.Info("Auto-detected ownership",
		zap.String("contract", addr),
		zap.String("owner", owner))

	return pr.Load(ctx)
}

// PeriodicReload 定期重新加载权限表
func (pr *PrivilegeRegistry) PeriodicReload(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := pr.Load(ctx); err != nil {
				pr.logger.Error("Failed to reload privilege registry", zap.Error(err))
			}
		}
	}
}
