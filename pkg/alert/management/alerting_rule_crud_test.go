package management_test

import (
    "context"
    "strings"
    "testing"

    management "github.com/openshift/cluster-monitoring-operator/pkg/alert/management"
    monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func TestGetAlertingRule_Found(t *testing.T) {
    t.Parallel()

    m := &mockClient{
        existingPrometheusRules: []monv1.Rule{
            {Alert: "OtherAlert", Labels: map[string]string{"severity": "warning"}},
            {Alert: testAlertName, Labels: map[string]string{"severity": "critical"}},
        },
    }

    c := &management.ControllerImpl{Client: m}

    ctx := context.Background()
    arID := management.AlertingRuleId{
        Namespace:      testNamespace,
        PrometheusRule: "test-pr",
        RuleName:       testAlertName,
        Severity:       "critical",
    }

    rule, err := c.GetAlertingRule(ctx, arID, management.Params{})
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if rule == nil {
        t.Fatalf("expected a rule, got nil")
    }
    if rule.Alert != testAlertName {
        t.Fatalf("expected alert name %q, got %q", testAlertName, rule.Alert)
    }
    if got := rule.Labels["severity"]; got != "critical" {
        t.Fatalf("expected severity label %q, got %q", "critical", got)
    }
}

func TestGetAlertingRule_NotFoundInPrometheusRule(t *testing.T) {
    t.Parallel()

    m := &mockClient{
        existingPrometheusRules: []monv1.Rule{
            // Same alert name but different severity to ensure mismatch
            {Alert: testAlertName, Labels: map[string]string{"severity": "warning"}},
            // Different alert name
            {Alert: "AnotherAlert", Labels: map[string]string{"severity": "critical"}},
        },
    }

    c := &management.ControllerImpl{Client: m}

    ctx := context.Background()
    arID := management.AlertingRuleId{
        Namespace:      testNamespace,
        PrometheusRule: "test-pr",
        RuleName:       testAlertName,
        Severity:       "critical", // looking for critical, but only warning exists for this name
    }

    rule, err := c.GetAlertingRule(ctx, arID, management.Params{})
    if err == nil {
        t.Fatalf("expected error, got nil with rule=%v", rule)
    }
    if !strings.Contains(err.Error(), "not found in PrometheusRule") {
        t.Fatalf("unexpected error: %v", err)
    }
}
