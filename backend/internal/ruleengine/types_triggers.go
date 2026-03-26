package ruleengine

// RuleTriggers 触发条件
type RuleTriggers struct {
	Operator   string          `yaml:"operator" json:"operator"`
	Conditions []RuleCondition `yaml:"conditions" json:"conditions"`
}

// RuleCondition 单个条件
type RuleCondition struct {
	Type        string      `yaml:"type" json:"type"`
	Operator    string      `yaml:"operator" json:"operator"`
	Value       interface{} `yaml:"value" json:"value"`
	Target      string      `yaml:"target" json:"target"`
	Within      string      `yaml:"within" json:"within"`
	Pattern     string      `yaml:"pattern" json:"pattern"`
	Description string      `yaml:"description" json:"description"`
}

// RuleExtract 数据提取
type RuleExtract struct {
	Transaction  []ExtractField            `yaml:"transaction" json:"transaction"`
	CallStack    []ExtractField            `yaml:"call_stack" json:"call_stack"`
	StateChanges []ExtractField            `yaml:"state_changes" json:"state_changes"`
	Events       []ExtractEventField       `yaml:"events" json:"events"`
	Custom       map[string][]ExtractField `yaml:"custom" json:"custom"`
}

// ExtractField 提取字段
type ExtractField struct {
	Field string `yaml:"field" json:"field"`
	As    string `yaml:"as" json:"as"`
}

// ExtractEventField 提取事件字段
type ExtractEventField struct {
	Event  string         `yaml:"event" json:"event"`
	Fields []ExtractField `yaml:"fields" json:"fields"`
	As     string         `yaml:"as" json:"as"`
}
