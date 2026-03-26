package ruleengine

// RuleScoring 风险评分
type RuleScoring struct {
	BaseScore int           `yaml:"base_score" json:"base_score"`
	Factors   []ScoreFactor `yaml:"factors" json:"factors"`
}

// ScoreFactor 评分因子
type ScoreFactor struct {
	Condition   string `yaml:"condition" json:"condition"`
	Score       int    `yaml:"score" json:"score"`
	Description string `yaml:"description" json:"description"`
}

// RuleAction 执行动作
type RuleAction struct {
	Type       string                 `yaml:"type" json:"type"`
	Severity   string                 `yaml:"severity" json:"severity"`
	Title      string                 `yaml:"title" json:"title"`
	Message    string                 `yaml:"message" json:"message"`
	Metadata   map[string]interface{} `yaml:"metadata" json:"metadata"`
	Channels   []string               `yaml:"channels" json:"channels"`
	Recipients []string               `yaml:"recipients" json:"recipients"`
	Script     string                 `yaml:"script" json:"script"`
	Args       []string               `yaml:"args" json:"args"`
}

// RuleFilters 过滤器
type RuleFilters struct {
	Whitelist FilterList `yaml:"whitelist" json:"whitelist"`
	Blacklist FilterList `yaml:"blacklist" json:"blacklist"`
}

// FilterList 过滤列表
type FilterList struct {
	Contracts []string `yaml:"contracts" json:"contracts"`
	Addresses []string `yaml:"addresses" json:"addresses"`
}
