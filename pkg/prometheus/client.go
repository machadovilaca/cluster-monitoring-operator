// Copyright 2019 The Cluster Monitoring Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/Jeffail/gabs"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/openshift/cluster-monitoring-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Client provides access to the Prometheus, Thanos & Alertmanager API.
type Client struct {
	// Host address of the endpoint.
	host string
	// Bearer token to use for authentication.
	token string
	// RoundTripper to use for HTTP transactions.
	rt http.RoundTripper
}

// NewClientFromRoute creates a new Client for the Prometheus instance in the given namespace and route name.
func NewClientFromRoute(ctx context.Context, cmoClient *client.Client, namespace, routeName string) (*Client, error) {
	prometheusURL, err := cmoClient.GetRouteURL(ctx, &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: cmoClient.Namespace(),
		},
	})
	if err != nil {
		return nil, err
	}

	token, err := GetServiceAccountToken(cmoClient, cmoClient.Namespace(), routeName)
	if err != nil {
		return nil, err
	}

	return NewClientFromHostToken(prometheusURL.Host, token), nil
}

// NewClientFromHostToken creates and returns a new Client with the given host and bearer token.
func NewClientFromHostToken(host, token string, wts ...WrapTransporter) *Client {
	// #nosec
	var rt http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	rt = (&HeaderInjector{Name: "Authorization", Value: "Bearer " + token}).WrapTransport(rt)
	rt = (&HeaderInjector{Name: "Content-Type", Value: "application/json"}).WrapTransport(rt)
	for i := range wts {
		rt = wts[i].WrapTransport(rt)
	}
	return &Client{
		host: host,
		rt:   rt,
	}
}

// WrapTransporter wraps an http.RoundTripper with another.
type WrapTransporter interface {
	WrapTransport(rt http.RoundTripper) http.RoundTripper
}

// MaxLength is the maximum string length returned by ClampMax().
const MaxLength = 1000

// ClampMax converts a slice of bytes to a string truncated to MaxLength.
func ClampMax(b []byte) string {
	s := string(b)
	if len(s) <= MaxLength {
		return s
	}
	return s[0:MaxLength-3] + "..."
}

// Do sends an HTTP request to the remote endpoint and returns the response.
func (c *Client) Do(method string, path string, body []byte) (*http.Response, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u.Host = c.host
	u.Scheme = "https"

	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return (&http.Client{Transport: c.rt}).Do(req)
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// HeaderInjector injects a fixed HTTP header into the inbound request.
type HeaderInjector struct {
	Name  string
	Value string
}

// WrapTransport implements the WrapTransporter interface.
func (h *HeaderInjector) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(
		func(req *http.Request) (*http.Response, error) {
			req.Header.Add(h.Name, h.Value)
			return rt.RoundTrip(req)
		},
	)
}

// QueryParameterInjector injects a fixed query parameter into the inbound request.
// It is typically used when querying kube-rbac-proxy.
type QueryParameterInjector struct {
	Name  string
	Value string
}

// WrapTransport implements the WrapTransporter interface.
func (qp *QueryParameterInjector) WrapTransport(rt http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(
		func(req *http.Request) (*http.Response, error) {
			q := req.URL.Query()
			q.Add(qp.Name, qp.Value)
			req.URL.RawQuery = q.Encode()
			return rt.RoundTrip(req)
		},
	)
}

// PrometheusQuery runs an HTTP GET request against the Prometheus query API and returns
// the response body.
func (c *Client) PrometheusQuery(query string) ([]byte, error) {
	return c.PrometheusQueryWithStatus(query, http.StatusOK)
}

func (c *Client) PrometheusQueryWithStatus(query string, status int) ([]byte, error) {
	resp, err := c.Do("GET", fmt.Sprintf("/api/v1/query?query=%s", url.QueryEscape(query)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != status {
		return nil, fmt.Errorf("unexpected status code response, want %d, got %d (%q)", status, resp.StatusCode, ClampMax(body))
	}

	return body, nil
}

// PrometheusTargets runs an HTTP GET request against the Prometheus targets API and returns
// the response body.
func (c *Client) PrometheusTargets() ([]byte, error) {
	resp, err := c.Do("GET", "/api/v1/targets", nil)
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

	return body, nil
}

// PrometheusRules runs an HTTP GET request against the Prometheus rules API and returns
// the response body.
func (c *Client) PrometheusRules() ([]byte, error) {
	resp, err := c.Do("GET", "/api/v1/rules", nil)
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

	return body, nil
}

// PrometheusLabel runs an HTTP GET request against the Prometheus label API and returns
// the response body.
func (c *Client) PrometheusLabel(label string) ([]byte, error) {
	resp, err := c.Do("GET", fmt.Sprintf("/api/v1/label/%s/values", url.QueryEscape(label)), nil)
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

	return body, nil
}

// GetAlertmanagerAlerts runs an HTTP GET request against the Alertmanager
// /api/v2/alerts endpoint and returns the response body.
func (c *Client) GetAlertmanagerAlerts(kvs ...string) ([]byte, error) {
	return c.getAlertmanager("/api/v2/alerts", kvs...)
}

// GetAlertmanagerSilences runs an HTTP GET request against the Alertmanager
// /api/v2/silences endpoint and returns the response body.
func (c *Client) GetAlertmanagerSilences(kvs ...string) ([]byte, error) {
	return c.getAlertmanager("/api/v2/silences", kvs...)
}

func (c *Client) getAlertmanager(path string, kvs ...string) ([]byte, error) {
	q := make(url.Values)
	for i := 0; i < len(kvs)/2; i++ {
		q.Add(kvs[i*2], kvs[i*2+1])
	}
	u := url.URL{
		Path:     path,
		RawQuery: q.Encode(),
	}

	resp, err := c.Do("GET", u.String(), nil)
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
		return nil, fmt.Errorf("%s: unexpected status code response, want %d, got %d (%q)", path, http.StatusOK, resp.StatusCode, ClampMax(body))
	}

	return body, nil
}

// GetFirstValueFromPromQuery takes a query api response body and returns the
// value of the first timeseries. If body contains multiple timeseries
// GetFirstValueFromPromQuery errors.
func GetFirstValueFromPromQuery(body []byte) (float64, error) {
	res, err := gabs.ParseJSON(body)
	if err != nil {
		return 0, err
	}

	count, err := res.ArrayCountP("data.result")
	if err != nil {
		return 0, err
	}

	if count != 1 {
		return 0, fmt.Errorf("expected body to contain single timeseries but got %v", count)
	}

	timeseries, err := res.ArrayElementP(0, "data.result")
	if err != nil {
		return 0, err
	}

	value, err := timeseries.ArrayElementP(1, "value")
	if err != nil {
		return 0, err
	}

	v, err := strconv.ParseFloat(value.Data().(string), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse query value: %w", err)
	}

	return v, nil
}

// GetResultSizeFromPromQuery takes a query api response body and returns the
// size of the result vector.
func GetResultSizeFromPromQuery(body []byte) (int, error) {
	res, err := gabs.ParseJSON(body)
	if err != nil {
		return 0, err
	}

	count, err := res.ArrayCountP("data.result")
	if err != nil {
		return 0, err
	}

	return count, nil
}

// WaitForQueryReturnGreaterEqualOne see WaitForQueryReturn.
func (c *Client) WaitForQueryReturnGreaterEqualOne(t *testing.T, timeout time.Duration, query string) {
	t.Helper()

	c.WaitForQueryReturn(t, timeout, query, func(v float64) error {
		if v >= 1 {
			return nil
		}

		return fmt.Errorf("expected value to equal or greater than 1 but got %v", v)
	})
}

// WaitForQueryReturnOne see WaitForQueryReturn.
func (c *Client) WaitForQueryReturnOne(t *testing.T, timeout time.Duration, query string) {
	t.Helper()

	c.WaitForQueryReturn(t, timeout, query, func(v float64) error {
		if v == 1 {
			return nil
		}

		return fmt.Errorf("expected value to equal 1 but got %v", v)
	})
}

// WaitForQueryReturn waits for a given PromQL query for a given time interval
// and validates the **first and only** result with the given validate function.
func (c *Client) WaitForQueryReturn(t *testing.T, timeout time.Duration, query string, validate func(float64) error) {
	t.Helper()

	err := Poll(5*time.Second, timeout, func() error {
		body, err := c.PrometheusQuery(query)
		if err != nil {
			return fmt.Errorf("error getting response for query %q: %w", query, err)
		}

		v, err := GetFirstValueFromPromQuery(body)
		if err != nil {
			return fmt.Errorf("error getting first value from response body %q for query %q: %w", string(body), query, err)
		}

		if err := validate(v); err != nil {
			return fmt.Errorf("error validating response body %q for query %q: %w", string(body), query, err)
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

// WaitForQueryReturnEmpty waits for a given PromQL query return an empty response for a given time interval
func (c *Client) WaitForQueryReturnEmpty(t *testing.T, timeout time.Duration, query string) {
	t.Helper()

	err := Poll(5*time.Second, timeout, func() error {
		body, err := c.PrometheusQuery(query)
		if err != nil {
			return fmt.Errorf("error getting response for query %q: %w", query, err)
		}

		size, err := GetResultSizeFromPromQuery(body)
		if err != nil {
			return fmt.Errorf("error getting body size from body %q for query %q: %w", string(body), query, err)
		}

		if size > 0 {
			return fmt.Errorf("expecting empty response but get %d results for query %s", size, query)
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

// WaitForRulesReturn waits for Prometheus rules for a given time interval
// and validates the **first and only** result with the given validate function.
func (c *Client) WaitForRulesReturn(t *testing.T, timeout time.Duration, validate func([]byte) error) {
	t.Helper()

	err := Poll(5*time.Second, timeout, func() error {
		body, err := c.PrometheusRules()
		if err != nil {
			return fmt.Errorf("error getting rules: %w", err)
		}

		if err := validate(body); err != nil {
			return fmt.Errorf("error validating response body %q: %w", string(body), err)
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

// WaitForTargetsReturn waits for Prometheus targets for a given time interval
// and returns successfully if the validate function doesn't return an error.
func (c *Client) WaitForTargetsReturn(t *testing.T, timeout time.Duration, validate func([]byte) error) {
	t.Helper()

	err := Poll(5*time.Second, timeout, func() error {
		body, err := c.PrometheusTargets()
		if err != nil {
			return fmt.Errorf("error getting targets: %w", err)
		}

		if err := validate(body); err != nil {
			return fmt.Errorf("error validating response body %q: %w", string(body), err)
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

// Poll calls the given function f every given interval
// until it returns no error or the given timeout occurs.
// If a timeout occurs, the last observed error is returned
// or wait.ErrWaitTimeout if no error occurred.
func Poll(interval, timeout time.Duration, f func() error) error {
	var lastErr error

	err := wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, func(context.Context) (bool, error) {
		lastErr = f()
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		if wait.Interrupted(err) && lastErr != nil {
			err = fmt.Errorf("%w: %w", err, lastErr)
		}
	}

	return err
}
