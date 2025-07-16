package management_test

import (
	"context"

	osmv1 "github.com/openshift/api/monitoring/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace = "openshift-monitoring"
	testAlertName = "TestAlert"
)

// mockClient implements the minimal interface needed for testing
type mockClient struct {
	existingRelabelConfig   []osmv1.RelabelConfig
	existingPrometheusRules []monv1.Rule

	createOrUpdateAlertRelabelConfigCalled bool
	deleteRelabelConfigError               error
	deleteRelabelConfigCalled              bool

	createOrUpdatePrometheusRuleCalled bool
	deletePrometheusRuleError          error
	deletePrometheusRuleCalled         bool
}

func (m *mockClient) Namespace() string {
	return testNamespace
}

func (m *mockClient) ListPrometheusRules(ctx context.Context) ([]monv1.PrometheusRule, error) {
	return []monv1.PrometheusRule{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testAlertName,
				Namespace: testNamespace,
				Labels: map[string]string{
					"cmo.openshift.io/owner": "alert-management",
				},
			},
			Spec: monv1.PrometheusRuleSpec{
				Groups: []monv1.RuleGroup{
					{
						Name:  "test-group",
						Rules: m.existingPrometheusRules,
					},
				},
			},
		},
	}, nil
}

func (m *mockClient) GetPrometheusRule(ctx context.Context, namespace, name string) (*monv1.PrometheusRule, error) {
	return &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cmo.openshift.io/owner": "alert-management",
			},
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{
				{
					Name:  "test-group",
					Rules: m.existingPrometheusRules,
				},
			},
		},
	}, nil
}

func (m *mockClient) CreateOrUpdatePrometheusRule(ctx context.Context, pr *monv1.PrometheusRule) error {
	m.createOrUpdatePrometheusRuleCalled = true
	return nil
}

func (m *mockClient) DeletePrometheusRuleByNamespaceAndName(ctx context.Context, namespace, name string) error {
	m.deletePrometheusRuleCalled = true
	return m.deletePrometheusRuleError
}

func (m *mockClient) ListAlertRelabelConfigs(ctx context.Context) ([]osmv1.AlertRelabelConfig, error) {
	return []osmv1.AlertRelabelConfig{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testAlertName,
				Namespace: testNamespace,
				Labels: map[string]string{
					"cmo.openshift.io/owner": "alert-management",
				},
			},
			Spec: osmv1.AlertRelabelConfigSpec{
				Configs: m.existingRelabelConfig,
			},
		},
	}, nil
}

func (m *mockClient) GetAlertRelabelConfig(ctx context.Context, namespace, name string) (*osmv1.AlertRelabelConfig, error) {
	return &osmv1.AlertRelabelConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: osmv1.AlertRelabelConfigSpec{
			Configs: m.existingRelabelConfig,
		},
	}, nil
}

func (m *mockClient) CreateOrUpdateAlertRelabelConfig(ctx context.Context, arc *osmv1.AlertRelabelConfig) error {
	m.createOrUpdateAlertRelabelConfigCalled = true
	return nil
}

func (m *mockClient) DeleteAlertRelabelConfigByNamespaceAndName(ctx context.Context, namespace, name string) error {
	m.deleteRelabelConfigCalled = true
	return m.deleteRelabelConfigError
}
