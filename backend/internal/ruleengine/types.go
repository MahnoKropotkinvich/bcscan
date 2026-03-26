package ruleengine

import "time"

// Rule 规则定义
type Rule struct {
	Metadata RuleMetadata `yaml:"metadata" json:"metadata"`
	Config   RuleConfig   `yaml:"config"   json:"config"`
	Triggers RuleTriggers `yaml:"triggers" json:"triggers"`
	Extract  RuleExtract  `yaml:"extract"  json:"extract"`
	Scoring  RuleScoring  `yaml:"scoring"  json:"scoring"`
	Actions  []RuleAction `yaml:"actions"  json:"actions"`
	Filters  RuleFilters  `yaml:"filters"  json:"filters"`
}

// RuleMetadata 规则元数据
type RuleMetadata struct {
	Name        string    `yaml:"name"        json:"name"`
	Version     string    `yaml:"version"     json:"version"`
	Author      string    `yaml:"author"      json:"author"`
	Description string    `yaml:"description" json:"description"`
	Tags        []string  `yaml:"tags"        json:"tags"`
	Enabled     bool      `yaml:"enabled"     json:"enabled"`
	CreatedAt   time.Time `yaml:"created_at"  json:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"  json:"updated_at"`
}

// RuleConfig 规则配置
type RuleConfig struct {
	Severity string         `yaml:"severity" json:"severity"`
	Priority int            `yaml:"priority" json:"priority"`
	Throttle ThrottleConfig `yaml:"throttle" json:"throttle"`
	Hooks    []string       `yaml:"hooks"    json:"hooks"`
}

// ThrottleConfig 限流配置
type ThrottleConfig struct {
	Enabled    bool   `yaml:"enabled"     json:"enabled"`
	MaxAlerts  int    `yaml:"max_alerts"  json:"max_alerts"`
	TimeWindow string `yaml:"time_window" json:"time_window"`
}
