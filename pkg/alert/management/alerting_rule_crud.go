package management

import (
	"context"
	"errors"
	"fmt"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

type AlertingRuleCRUD interface {
	GetAlertingRule(ctx context.Context, arID AlertingRuleId, params Params) (*monv1.Rule, error)
	CreateAlertingRule(ctx context.Context, arID AlertingRuleId, rule monv1.Rule, params Params) (*monv1.Rule, error)
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

func (c *ControllerImpl) CreateAlertingRule(ctx context.Context, arID AlertingRuleId, rule monv1.Rule, _ Params) (*monv1.Rule, error) {
	prometheusRule, found, err := c.getPrometheusRule(ctx, arID.Namespace, arID.PrometheusRule)
	if err != nil {
		return nil, fmt.Errorf("unexpected error getting PrometheusRule %s/%s", arID.Namespace, arID.PrometheusRule)
	}

	if found && !isCMOManagedPrometheusRule(prometheusRule) {
		return nil, fmt.Errorf("PrometheusRule %s/%s is not managed by CMO", arID.Namespace, arID.PrometheusRule)
	}

	var ruleGroup *monv1.RuleGroup

	if found {
		ruleGroup, err = findCMOManagedRuleGroup(prometheusRule)
		if err != nil {
			return nil, err
		}
	} else {
		ruleGroup = &monv1.RuleGroup{
			Name:  PrometheusRuleGroupName,
			Rules: []monv1.Rule{},
		}
	}

	ruleGroup.Rules = append(ruleGroup.Rules, rule)

	err = c.savePrometheusRule(ctx, arID.Namespace, arID.PrometheusRule, ruleGroup.Rules)
	if err != nil {
		return nil, fmt.Errorf("unexpected error saving PrometheusRule %s/%s", arID.Namespace, arID.PrometheusRule)
	}

	return &rule, nil
}

func findCMOManagedRuleGroup(pr *monv1.PrometheusRule) (*monv1.RuleGroup, error) {
	// Find the rule group with the name "cmo-alert-management"
	for i, group := range pr.Spec.Groups {
		if group.Name == PrometheusRuleGroupName {
			return &pr.Spec.Groups[i], nil
		}
	}

	return nil, errors.New("CMO managed rule group not found")
}
