package ruleengine

import (
	"fmt"

	"go.uber.org/zap"
)

type Engine struct {
	loader *RuleLoader
	logger *zap.Logger
}

func NewEngine(rulesDir string, logger *zap.Logger) (*Engine, error) {
	loader := NewRuleLoader(rulesDir, logger)

	if err := loader.LoadAll(); err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	return &Engine{
		loader: loader,
		logger: logger,
	}, nil
}

func (e *Engine) GetEnabledRules() []*Rule {
	return e.loader.GetEnabledRules()
}
