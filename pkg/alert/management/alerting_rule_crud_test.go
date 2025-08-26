package management_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/openshift/cluster-monitoring-operator/pkg/alert/management"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// --- GetAlertingRule tests ---

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

// --- CreateAlertingRule tests ---

func TestCreateAlertingRule_NewPR_CreatesRule(t *testing.T) {
	t.Parallel()

	m := &mockClient{getPRReturnNotFound: true}
	c := &management.ControllerImpl{Client: m}

	ctx := context.Background()
	arID := management.AlertingRuleId{
		Namespace:      testNamespace,
		PrometheusRule: "test-pr",
		RuleName:       testAlertName,
		Severity:       "critical",
	}

	newRule := monv1.Rule{Alert: testAlertName, Labels: map[string]string{"severity": "critical"}}

	got, err := c.CreateAlertingRule(ctx, arID, newRule, management.Params{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.Alert != testAlertName {
		t.Fatalf("unexpected created rule: %#v", got)
	}
	if !m.createOrUpdatePrometheusRuleCalled {
		t.Fatalf("expected CreateOrUpdatePrometheusRule to be called")
	}
}

func TestCreateAlertingRule_ExistingPRWithCMOGroup_AppendsAndSaves(t *testing.T) {
	t.Parallel()

	m := &mockClient{getPRHasCMOGroup: true, existingPrometheusRules: []monv1.Rule{{Alert: "Other", Labels: map[string]string{"severity": "warning"}}}}
	c := &management.ControllerImpl{Client: m}

	ctx := context.Background()
	arID := management.AlertingRuleId{
		Namespace:      testNamespace,
		PrometheusRule: "test-pr",
		RuleName:       testAlertName,
		Severity:       "critical",
	}

	newRule := monv1.Rule{Alert: testAlertName, Labels: map[string]string{"severity": "critical"}}

	got, err := c.CreateAlertingRule(ctx, arID, newRule, management.Params{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.Alert != testAlertName {
		t.Fatalf("unexpected created rule: %#v", got)
	}
	if !m.createOrUpdatePrometheusRuleCalled {
		t.Fatalf("expected CreateOrUpdatePrometheusRule to be called")
	}
}

func TestCreateAlertingRule_ExistingPR_NoCMOGroup_ReturnsError(t *testing.T) {
	t.Parallel()

	m := &mockClient{}
	c := &management.ControllerImpl{Client: m}

	ctx := context.Background()
	arID := management.AlertingRuleId{
		Namespace:      testNamespace,
		PrometheusRule: "test-pr",
		RuleName:       testAlertName,
		Severity:       "critical",
	}

	newRule := monv1.Rule{Alert: testAlertName, Labels: map[string]string{"severity": "critical"}}

	got, err := c.CreateAlertingRule(ctx, arID, newRule, management.Params{})
	if err == nil {
		t.Fatalf("expected error, got nil (rule=%#v)", got)
	}
	if !strings.Contains(err.Error(), "CMO managed rule group not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.createOrUpdatePrometheusRuleCalled {
		t.Fatalf("did not expect CreateOrUpdatePrometheusRule to be called")
	}
}

func TestCreateAlertingRule_ExistingPR_NotManaged_ReturnsError(t *testing.T) {
	t.Parallel()

	m := &mockClient{getPRNotManaged: true}
	c := &management.ControllerImpl{Client: m}

	ctx := context.Background()
	arID := management.AlertingRuleId{
		Namespace:      testNamespace,
		PrometheusRule: "test-pr",
		RuleName:       testAlertName,
		Severity:       "critical",
	}

	newRule := monv1.Rule{Alert: testAlertName, Labels: map[string]string{"severity": "critical"}}

	got, err := c.CreateAlertingRule(ctx, arID, newRule, management.Params{})
	if err == nil {
		t.Fatalf("expected error, got nil (rule=%#v)", got)
	}
	if !strings.Contains(err.Error(), "not managed by CMO") {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.createOrUpdatePrometheusRuleCalled {
		t.Fatalf("did not expect CreateOrUpdatePrometheusRule to be called")
	}
}

func TestCreateAlertingRule_GetPRUnexpectedError(t *testing.T) {
	t.Parallel()

	m := &mockClient{getPRReturnErr: fmt.Errorf("boom")}
	c := &management.ControllerImpl{Client: m}

	ctx := context.Background()
	arID := management.AlertingRuleId{
		Namespace:      testNamespace,
		PrometheusRule: "test-pr",
		RuleName:       testAlertName,
		Severity:       "critical",
	}

	newRule := monv1.Rule{Alert: testAlertName, Labels: map[string]string{"severity": "critical"}}

	got, err := c.CreateAlertingRule(ctx, arID, newRule, management.Params{})
	if err == nil {
		t.Fatalf("expected error, got nil (rule=%#v)", got)
	}
	if !strings.Contains(err.Error(), "unexpected error getting PrometheusRule") {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.createOrUpdatePrometheusRuleCalled {
		t.Fatalf("did not expect CreateOrUpdatePrometheusRule to be called")
	}
}
