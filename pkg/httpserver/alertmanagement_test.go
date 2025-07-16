package httpserver_test

import (
	"net/http/httptest"
	"testing"

	"github.com/openshift/cluster-monitoring-operator/pkg/httpserver"
)

func TestParseAlertingRuleId_OK(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	// Manually set path values
	r.SetPathValue("namespace", "ns1")
	r.SetPathValue("prometheusrule", "pr1")
	r.SetPathValue("ruleName", "ruleA")
	r.SetPathValue("severity", "warning")

	id, err := httpserver.ParseAlertingRuleId(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Namespace != "ns1" || id.PrometheusRule != "pr1" || id.RuleName != "ruleA" || id.Severity != "warning" {
		t.Fatalf("unexpected id: %+v", id)
	}
}

func TestParseAlertingRuleId_Missing(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	// Intentionally missing severity
	r.SetPathValue("namespace", "ns1")
	r.SetPathValue("prometheusrule", "pr1")
	r.SetPathValue("ruleName", "ruleA")
	// severity not set -> empty

	_, err := httpserver.ParseAlertingRuleId(r)
	if err == nil {
		t.Fatalf("expected error for missing parameters, got nil")
	}
}
