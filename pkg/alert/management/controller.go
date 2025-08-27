package management

import (
	"context"
	"errors"

	osmv1 "github.com/openshift/api/monitoring/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/openshift/cluster-monitoring-operator/pkg/client"
	"github.com/openshift/cluster-monitoring-operator/pkg/prometheus"
)

const (
	ResourceOwnerLabelKey   = "cmo.openshift.io/owner"
	ResourceOwnerLabelValue = "alert-management"

	PrometheusRuleGroupName = "cmo-alert-management"
)

type Controller interface {
	AlertingRuleCRUD
}

type ControllerImpl struct {
	Client           Client
	PrometheusClient PrometheusClient
}

func NewController(ctx context.Context, client *client.Client, serverAddr string) (Controller, error) {
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}

	prometheusClient, err := prometheus.NewClientFromRoute(ctx, client, client.Namespace(), "prometheus-k8s")
	if err != nil {
		return nil, err
	}

	return &ControllerImpl{
		Client:           client,
		PrometheusClient: prometheusClient,
	}, nil
}

// private

func (c *ControllerImpl) getPrometheusRule(ctx context.Context, namespace string, name string) (*monv1.PrometheusRule, bool, error) {
	pr, err := c.Client.GetPrometheusRule(ctx, namespace, name)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("error getting PrometheusRule %s/%s: %v", namespace, name, err)
		return nil, false, err
	}

	if pr == nil || pr.Name == "" {
		return nil, false, nil
	}

	return pr, true, nil
}

func isCMOManagedPrometheusRule(pr *monv1.PrometheusRule) bool {
	if val, ok := pr.Labels[ResourceOwnerLabelKey]; !ok || val != ResourceOwnerLabelValue {
		return false
	}

	return true
}

func (c *ControllerImpl) savePrometheusRule(ctx context.Context, namespace string, name string, rules []monv1.Rule) error {
	pr, found, err := c.getPrometheusRule(ctx, namespace, name)
	if err != nil {
		klog.Errorf("error getting PrometheusRule %s/%s: %v", namespace, name, err)
		return err
	}

	if found && !isCMOManagedPrometheusRule(pr) {
		return errors.New("PrometheusRule already exists and is not managed by CMO alert management")
	}

	if len(rules) == 0 {
		if found {
			return c.Client.DeletePrometheusRuleByNamespaceAndName(ctx, namespace, name)
		} else {
			return nil
		}
	}

	newPR := &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ResourceOwnerLabelKey: ResourceOwnerLabelValue,
			},
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{
				{
					Name:  "cmo-alert-management",
					Rules: rules,
				},
			},
		},
	}

	return c.Client.CreateOrUpdatePrometheusRule(ctx, newPR)
}

func (c *ControllerImpl) saveAlertRelabelConfig(ctx context.Context, namespace string, name string, config []osmv1.RelabelConfig) error {
	relabelConfig, err := c.Client.GetAlertRelabelConfig(ctx, namespace, name)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if relabelConfig != nil {
		// If the AlertRelabelConfig already exists
		// Check if it has the cmo.openshift.io/owner label
		if val, ok := relabelConfig.Labels[ResourceOwnerLabelKey]; !ok || val != ResourceOwnerLabelValue {
			return errors.New("AlertRelabelConfig already exists and is not managed by CMO alert management")
		}
	}

	if len(config) == 0 {
		return c.Client.DeleteAlertRelabelConfigByNamespaceAndName(ctx, namespace, name)
	}

	newRC := &osmv1.AlertRelabelConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				ResourceOwnerLabelKey: ResourceOwnerLabelValue,
			},
		},
		Spec: osmv1.AlertRelabelConfigSpec{
			Configs: config,
		},
	}

	return c.Client.CreateOrUpdateAlertRelabelConfig(ctx, newRC)
}
