package management

import (
	"context"

	osmv1 "github.com/openshift/api/monitoring/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/openshift/cluster-monitoring-operator/pkg/prometheus"
)

type Client interface {
	Namespace() string

	ListPrometheusRules(ctx context.Context) ([]monv1.PrometheusRule, error)
	GetPrometheusRule(ctx context.Context, namespace, name string) (*monv1.PrometheusRule, error)
	CreateOrUpdatePrometheusRule(ctx context.Context, rule *monv1.PrometheusRule) error
	DeletePrometheusRuleByNamespaceAndName(ctx context.Context, namespace, name string) error

	ListAlertRelabelConfigs(ctx context.Context) ([]osmv1.AlertRelabelConfig, error)
	GetAlertRelabelConfig(ctx context.Context, namespace, name string) (*osmv1.AlertRelabelConfig, error)
	CreateOrUpdateAlertRelabelConfig(ctx context.Context, relabelConfig *osmv1.AlertRelabelConfig) error
	DeleteAlertRelabelConfigByNamespaceAndName(ctx context.Context, namespace, name string) error
}

type PrometheusClient interface {
	ListAlertingRules(name string) ([]prometheus.Rule, error)
}
