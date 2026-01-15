package ruleengine

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type RuleLoader struct {
	rulesDir string
	logger   *zap.Logger
	rules    map[string]*Rule
}

func NewRuleLoader(rulesDir string, logger *zap.Logger) *RuleLoader {
	return &RuleLoader{
		rulesDir: rulesDir,
		logger:   logger,
		rules:    make(map[string]*Rule),
	}
}

func (rl *RuleLoader) LoadAll() error {
	pattern := filepath.Join(rl.rulesDir, "**/*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob rules: %w", err)
	}

	// Also check direct yaml files in rules directory
	directFiles, err := filepath.Glob(filepath.Join(rl.rulesDir, "*.yaml"))
	if err == nil {
		files = append(files, directFiles...)
	}

	rl.logger.Info("Loading rules", zap.Int("file_count", len(files)))

	for _, file := range files {
		if err := rl.LoadFile(file); err != nil {
			rl.logger.Error("Failed to load rule file",
				zap.String("file", file),
				zap.Error(err))
			continue
		}
	}

	rl.logger.Info("Rules loaded successfully", zap.Int("rule_count", len(rl.rules)))
	return nil
}

func (rl *RuleLoader) LoadFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var rule Rule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	if rule.Metadata.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	rl.rules[rule.Metadata.Name] = &rule
	rl.logger.Info("Rule loaded",
		zap.String("name", rule.Metadata.Name),
		zap.String("file", filename))

	return nil
}

func (rl *RuleLoader) GetRule(name string) (*Rule, bool) {
	rule, ok := rl.rules[name]
	return rule, ok
}

func (rl *RuleLoader) GetAllRules() map[string]*Rule {
	return rl.rules
}

func (rl *RuleLoader) GetEnabledRules() []*Rule {
	enabled := make([]*Rule, 0)
	for _, rule := range rl.rules {
		if rule.Metadata.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}
