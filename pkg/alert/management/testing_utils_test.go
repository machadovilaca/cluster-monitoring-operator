package management_test

import (
	"context"

	osmv1 "github.com/openshift/api/monitoring/v1"
	"github.com/openshift/cluster-monitoring-operator/pkg/alert/management"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	testNamespace = "openshift-monitoring"
	testAlertName = "TestAlert"
)

// mockClient implements the minimal interface needed for testing
type mockClient struct {
	existingRelabelConfig   []osmv1.RelabelConfig
	existingPrometheusRules []monv1.Rule

	// Behavior flags for PrometheusRule retrieval
	getPRReturnNotFound bool
	getPRReturnErr      error
	getPRNotManaged     bool
	getPRHasCMOGroup    bool

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
					management.ResourceOwnerLabelKey: management.ResourceOwnerLabelValue,
				},
			},
			Spec: monv1.PrometheusRuleSpec{
				Groups: []monv1.RuleGroup{
					{
						Name:  management.PrometheusRuleGroupName,
						Rules: m.existingPrometheusRules,
					},
				},
			},
		},
	}, nil
}

func (m *mockClient) GetPrometheusRule(ctx context.Context, namespace, name string) (*monv1.PrometheusRule, error) {
	if m.getPRReturnErr != nil {
		return nil, m.getPRReturnErr
	}
	if m.getPRReturnNotFound {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "monitoring.coreos.com", Resource: "prometheusrules"}, name)
	}

	labels := map[string]string{}
	if m.getPRNotManaged {
		labels[management.ResourceOwnerLabelKey] = "someone-else"
	} else {
		labels[management.ResourceOwnerLabelKey] = management.ResourceOwnerLabelValue
	}

	groupName := "test-group"
	if m.getPRHasCMOGroup {
		groupName = management.PrometheusRuleGroupName
	}

	return &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{
				{
					Name:  groupName,
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
					management.ResourceOwnerLabelKey: management.ResourceOwnerLabelValue,
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
