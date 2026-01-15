package hooks

import (
	"fmt"
	"sync"

	"github.com/haswell/bcscan/internal/ruleengine"
)

// Manager Hook 管理器
type Manager struct {
	hooks map[string]Hook
	mu    sync.RWMutex
}

// NewManager 创建新的 Hook 管理器
func NewManager() *Manager {
	return &Manager{
		hooks: make(map[string]Hook),
	}
}

// Register 注册钩子
func (m *Manager) Register(hook Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks[hook.Name()] = hook
}

// Get 获取钩子
func (m *Manager) Get(name string) (Hook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hook, ok := m.hooks[name]
	if !ok {
		return nil, fmt.Errorf("hook not found: %s", name)
	}
	return hook, nil
}

// Trigger 触发指定钩子
func (m *Manager) Trigger(hookName string, ctx *ruleengine.EvaluationContext, rules []*ruleengine.Rule) ([]*RiskEvent, error) {
	hook, err := m.Get(hookName)
	if err != nil {
		return nil, err
	}

	return hook.Execute(ctx, rules)
}
