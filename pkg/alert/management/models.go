package management

type Params struct{}

type AlertingRuleId struct {
	Namespace      string
	PrometheusRule string
	RuleName       string
	Severity       string
}
