package ruleengine

// Scorer 风险评分器
type Scorer struct {
	evaluator *Evaluator
}

// NewScorer 创建新的评分器
func NewScorer() *Scorer {
	return &Scorer{
		evaluator: NewEvaluator(),
	}
}

// CalculateScore 计算风险分数
// 基础分数 + 动态因子分数
func (s *Scorer) CalculateScore(rule *Rule, ctx *EvaluationContext) (int, error) {
	// 从基础分数开始
	totalScore := rule.Scoring.BaseScore

	// 评估每个评分因子
	for _, factor := range rule.Scoring.Factors {
		matched, err := s.evaluator.Evaluate(factor.Condition, ctx)
		if err != nil {
			return 0, err
		}

		if matched {
			totalScore += factor.Score
		}
	}

	// 确保分数在 0-100 范围内
	if totalScore < 0 {
		totalScore = 0
	}
	if totalScore > 100 {
		totalScore = 100
	}

	return totalScore, nil
}
