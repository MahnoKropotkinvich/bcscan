package ruleengine

import "time"

// Rule 规则定义
type Rule struct {
	Metadata RuleMetadata `yaml:"metadata"`
	Config   RuleConfig   `yaml:"config"`
	Triggers RuleTriggers `yaml:"triggers"`
	Extract  RuleExtract  `yaml:"extract"`
	Scoring  RuleScoring  `yaml:"scoring"`
	Actions  []RuleAction `yaml:"actions"`
	Filters  RuleFilters  `yaml:"filters"`
}

// RuleMetadata 规则元数据
type RuleMetadata struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Author      string    `yaml:"author"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags"`
	Enabled     bool      `yaml:"enabled"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
}

// RuleConfig 规则配置
type RuleConfig struct {
	Severity string         `yaml:"severity"`
	Priority int            `yaml:"priority"`
	Throttle ThrottleConfig `yaml:"throttle"`
	Hooks    []string       `yaml:"hooks"`
}

// ThrottleConfig 限流配置
type ThrottleConfig struct {
	Enabled    bool   `yaml:"enabled"`
	MaxAlerts  int    `yaml:"max_alerts"`
	TimeWindow string `yaml:"time_window"`
}
