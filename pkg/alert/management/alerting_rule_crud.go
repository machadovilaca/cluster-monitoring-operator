package management

import (
	"context"
	"fmt"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

type AlertingRuleCRUD interface {
	GetAlertingRule(ctx context.Context, arID AlertingRuleId, params Params) (*monv1.Rule, error)
}

func (c *ControllerImpl) GetAlertingRule(ctx context.Context, arID AlertingRuleId, _ Params) (*monv1.Rule, error) {
	prometheusRule, err := c.Client.GetPrometheusRule(ctx, arID.Namespace, arID.PrometheusRule)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("PrometheusRule %s/%s not found", arID.Namespace, arID.PrometheusRule)
		}

		klog.Errorf("error getting PrometheusRule %s/%s: %v", arID.Namespace, arID.PrometheusRule, err)
		return nil, fmt.Errorf("unexpected error getting PrometheusRule %s/%s", arID.Namespace, arID.PrometheusRule)
	}

	if prometheusRule == nil {
		return nil, fmt.Errorf("PrometheusRule %s/%s not found", arID.Namespace, arID.PrometheusRule)
	}

	for _, group := range prometheusRule.Spec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert == arID.RuleName && rule.Labels["severity"] == arID.Severity {
				return &rule, nil
			}
		}
	}

	return nil, fmt.Errorf("alerting rule %s/%s not found in PrometheusRule %s/%s", arID.Severity, arID.RuleName, arID.Namespace, arID.PrometheusRule)
}
