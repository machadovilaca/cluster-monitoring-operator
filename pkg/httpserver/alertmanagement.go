package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	alertmanagement "github.com/openshift/cluster-monitoring-operator/pkg/alert/management"
)

const alertingRuleIdPath = "/namespaces/{namespace}/prometheusrules/{prometheusrule}/rules/{ruleName}/severities/{severity}"

type alertManagementMux struct {
	alertsManagementController alertmanagement.Controller
}

func AlertManagementMux(alertsManagementController alertmanagement.Controller) *http.ServeMux {
	mux := http.NewServeMux()

	amm := &alertManagementMux{
		alertsManagementController: alertsManagementController,
	}

	mux.HandleFunc("GET /alerts", amm.listAlertsHandler)

	mux.HandleFunc("GET /rules", amm.listAlertingRulesHandler)
	mux.HandleFunc("POST /rules", amm.createAlertingRuleHandler)
	mux.HandleFunc("DELETE /rules", amm.deleteAlertingRulesHandler)

	mux.HandleFunc("GET /rules"+alertingRuleIdPath, amm.getAlertingRuleHandler)
	mux.HandleFunc("PUT /rules"+alertingRuleIdPath, amm.updateAlertingRuleHandler)
	mux.HandleFunc("PATCH /rules"+alertingRuleIdPath, amm.updateAlertingRuleHandler)
	mux.HandleFunc("DELETE /rules"+alertingRuleIdPath, amm.deleteAlertingRuleHandler)

	mux.HandleFunc("GET /rules"+alertingRuleIdPath+"/labels", amm.getAlertingRuleLabelsHandler)

	return mux
}

func (amm *alertManagementMux) listAlertsHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (amm *alertManagementMux) listAlertingRulesHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (amm *alertManagementMux) createAlertingRuleHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (amm *alertManagementMux) deleteAlertingRulesHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (amm *alertManagementMux) deleteAlertingRuleHandler(w http.ResponseWriter, r *http.Request) {
	arId, err := ParseAlertingRuleId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = arId

	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (amm *alertManagementMux) getAlertingRuleHandler(w http.ResponseWriter, r *http.Request) {
	arId, err := ParseAlertingRuleId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = arId

	alertingRule, err := amm.alertsManagementController.GetAlertingRule(r.Context(), arId, alertmanagement.Params{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if alertingRule == nil {
		http.Error(w, "alerting rule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alertingRule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (amm *alertManagementMux) updateAlertingRuleHandler(w http.ResponseWriter, r *http.Request) {
	arId, err := ParseAlertingRuleId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = arId

	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func (amm *alertManagementMux) getAlertingRuleLabelsHandler(w http.ResponseWriter, r *http.Request) {
	arId, err := ParseAlertingRuleId(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = arId

	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

// ParseAlertingRuleId extracts path values into AlertingRuleId and validates presence.
func ParseAlertingRuleId(r *http.Request) (alertmanagement.AlertingRuleId, error) {
	arId := alertmanagement.AlertingRuleId{
		Namespace:      r.PathValue("namespace"),
		PrometheusRule: r.PathValue("prometheusrule"),
		RuleName:       r.PathValue("ruleName"),
		Severity:       r.PathValue("severity"),
	}
	if arId.Namespace == "" || arId.PrometheusRule == "" || arId.RuleName == "" || arId.Severity == "" {
		return arId, errors.New("missing required path parameters")
	}
	return arId, nil
}
