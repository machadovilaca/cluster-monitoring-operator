package prometheus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RulesResponse represents the response from the Prometheus rules API
type RulesResponse struct {
	Data   RulesData `json:"data"`
	Status string    `json:"status"`
}

// RulesData represents the data section of the response
type RulesData struct {
	Groups []RuleGroup `json:"groups"`
}

// RuleGroup represents a group of rules
type RuleGroup struct {
	EvaluationTime float64   `json:"evaluationTime"`
	File           string    `json:"file"`
	Interval       int       `json:"interval"`
	LastEvaluation time.Time `json:"lastEvaluation"`
	Limit          int       `json:"limit"`
	Name           string    `json:"name"`
	Rules          []Rule    `json:"rules"`
}

// Rule represents an individual rule
type Rule struct {
	Alerts         []Alert           `json:"alerts"`
	Annotations    map[string]string `json:"annotations"`
	Duration       int               `json:"duration"`
	EvaluationTime float64           `json:"evaluationTime"`
	Health         string            `json:"health"`
	KeepFiringFor  int               `json:"keepFiringFor"`
	Labels         map[string]string `json:"labels"`
	LastEvaluation time.Time         `json:"lastEvaluation"`
	Name           string            `json:"name"`
	Query          string            `json:"query"`
	State          string            `json:"state"`
	Type           string            `json:"type"`
}

// Alert represents an active alert
type Alert struct {
	ActiveAt    time.Time         `json:"activeAt,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	State       string            `json:"state,omitempty"`
	Value       string            `json:"value,omitempty"`
}

// ListAlertingRules runs an HTTP GET request against the Prometheus rules API and returns
// a list of all PrometheusRule from all groups.
func (c *Client) ListAlertingRules(alertname string) ([]Rule, error) {
	resp, err := c.Do("GET", "/api/v1/rules?type=alert&rule_name[]="+alertname, nil)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code response, want %d, got %d (%q)", http.StatusOK, resp.StatusCode, ClampMax(body))
	}

	var rulesResponse RulesResponse
	if err := json.Unmarshal(body, &rulesResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Flatten all rules from all groups into a single slice
	var allRules []Rule
	for _, group := range rulesResponse.Data.Groups {
		allRules = append(allRules, group.Rules...)
	}

	return allRules, nil
}
