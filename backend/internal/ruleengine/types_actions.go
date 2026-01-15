package ruleengine

// RuleScoring 风险评分
type RuleScoring struct {
	BaseScore int           `yaml:"base_score"`
	Factors   []ScoreFactor `yaml:"factors"`
}

// ScoreFactor 评分因子
type ScoreFactor struct {
	Condition   string `yaml:"condition"`
	Score       int    `yaml:"score"`
	Description string `yaml:"description"`
}

// RuleAction 执行动作
type RuleAction struct {
	Type       string                 `yaml:"type"`
	Severity   string                 `yaml:"severity"`
	Title      string                 `yaml:"title"`
	Message    string                 `yaml:"message"`
	Metadata   map[string]interface{} `yaml:"metadata"`
	Channels   []string               `yaml:"channels"`
	Recipients []string               `yaml:"recipients"`
	Script     string                 `yaml:"script"`
	Args       []string               `yaml:"args"`
}

// RuleFilters 过滤器
type RuleFilters struct {
	Whitelist FilterList `yaml:"whitelist"`
	Blacklist FilterList `yaml:"blacklist"`
}

// FilterList 过滤列表
type FilterList struct {
	Contracts []string `yaml:"contracts"`
	Addresses []string `yaml:"addresses"`
}
