package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultRulePrefix = "alerts-perf-"
)

var (
	promRuleGVR = schema.GroupVersionResource{Group: "monitoring.coreos.com", Version: "v1", Resource: "prometheusrules"}
)

func buildConfig() (*rest.Config, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if home, ok := os.LookupEnv("HOME"); ok {
		kc := home + "/.kube/config"
		if _, err := os.Stat(kc); err == nil {
			return clientcmd.BuildConfigFromFlags("", kc)
		}
	}
	return rest.InClusterConfig()
}

func createOrKeepPromRule(ctx context.Context, dyn dynamic.Interface, namespace, name, expr string) error {
	res := dyn.Resource(promRuleGVR).Namespace(namespace)
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "monitoring.coreos.com/v1",
		"kind":       "PrometheusRule",
		"metadata": map[string]any{
			"name": name,
		},
		"spec": map[string]any{
			"groups": []any{
				map[string]any{
					"name": name + "-group",
					"rules": []any{
						map[string]any{
							"alert":       name + "-alert",
							"expr":        expr,
							"labels":      map[string]any{"severity": "none"},
							"annotations": map[string]any{"summary": "perf test alert"},
						},
					},
				},
			},
		},
	}}
	_, err := res.Create(ctx, obj, metav1.CreateOptions{})
	if err == nil {
		return nil
	}
	return nil
}

func parseCounts(countsStr string, fallback int) []int {
	if countsStr == "" {
		return []int{fallback}
	}
	parts := strings.Split(countsStr, ",")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err == nil && v > 0 {
			res = append(res, v)
		}
	}
	if len(res) == 0 {
		return []int{fallback}
	}
	return res
}

func writeLine(path, line string) error {
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

func main() {
	var (
		ruleExpr  string
		namespace string
		nList     string
		outPath   string
	)
	flag.StringVar(&nList, "n", "", "comma-separated list of counts to test (e.g. 200,500,1000); defaults to 200 if empty")
	flag.StringVar(&ruleExpr, "expr", "up == 0", "PromQL expression to use in each alert rule")
	flag.StringVar(&namespace, "namespace", "", "existing namespace containing/for the rules (required)")
	flag.StringVar(&outPath, "out", "", "file to append results to (optional)")
	flag.Parse()

	if namespace == "" {
		panic("--namespace is required")
	}

	cfg, err := buildConfig()
	if err != nil {
		panic(err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	counts := parseCounts(nList, 200)
	for _, n := range counts {
		// Ensure rules exist
		for i := 0; i < n; i++ {
			ruleName := fmt.Sprintf("%s%d", defaultRulePrefix, i)
			if err := createOrKeepPromRule(ctx, dyn, namespace, ruleName, ruleExpr); err != nil {
				panic(fmt.Errorf("ensure rule %s/%s: %w", namespace, ruleName, err))
			}
		}

		indices := make([]int, n)
		for i := 0; i < n; i++ {
			indices[i] = i
		}
		rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

		start := time.Now()
		totalRules := 0
		for _, i := range indices {
			ruleName := fmt.Sprintf("%s%d", defaultRulePrefix, i)
			obj, err := dyn.Resource(promRuleGVR).Namespace(namespace).Get(ctx, ruleName, metav1.GetOptions{})
			if err != nil {
				panic(fmt.Errorf("get rule %s/%s: %w", namespace, ruleName, err))
			}
			groups, found, _ := unstructured.NestedSlice(obj.Object, "spec", "groups")
			if found {
				for _, g := range groups {
					grp, ok := g.(map[string]any)
					if !ok {
						continue
					}
					rules, ok := grp["rules"].([]any)
					if ok {
						totalRules += len(rules)
					}
				}
			}
		}
		elapsed := time.Since(start)
		line := fmt.Sprintf("Fetched %d PrometheusRule objects containing %d rules in %s (avg %.2f ms per GET)", n, totalRules, elapsed.String(), float64(elapsed.Milliseconds())/float64(n))
		fmt.Println(line)
		if err := writeLine(outPath, line); err != nil {
			panic(err)
		}
	}
}
