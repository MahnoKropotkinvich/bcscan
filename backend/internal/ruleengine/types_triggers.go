package ruleengine

// RuleTriggers 触发条件
type RuleTriggers struct {
	Operator   string          `yaml:"operator"`
	Conditions []RuleCondition `yaml:"conditions"`
}

// RuleCondition 单个条件
type RuleCondition struct {
	Type        string      `yaml:"type"`
	Operator    string      `yaml:"operator"`
	Value       interface{} `yaml:"value"`
	Target      string      `yaml:"target"`
	Within      string      `yaml:"within"`
	Pattern     string      `yaml:"pattern"`
	Description string      `yaml:"description"`
}

// RuleExtract 数据提取
type RuleExtract struct {
	Transaction  []ExtractField            `yaml:"transaction"`
	CallStack    []ExtractField            `yaml:"call_stack"`
	StateChanges []ExtractField            `yaml:"state_changes"`
	Events       []ExtractEventField       `yaml:"events"`
	Custom       map[string][]ExtractField `yaml:"custom"`
}

// ExtractField 提取字段
type ExtractField struct {
	Field string `yaml:"field"`
	As    string `yaml:"as"`
}

// ExtractEventField 提取事件字段
type ExtractEventField struct {
	Event  string         `yaml:"event"`
	Fields []ExtractField `yaml:"fields"`
	As     string         `yaml:"as"`
}
