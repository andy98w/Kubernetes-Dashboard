// kubevista is an HTTP client for the same review and execution path as the UI.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	cluster "github.com/andy98w/Kubernetes-Dashboard/api/internal/kubernetes"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kubevista diagnose|plan|execute|verify|history [flags]")
	}
	command := args[0]
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	api := fs.String("api", "http://127.0.0.1:8080", "KubeVista API URL")
	namespace := fs.String("namespace", "kubevista-lab", "workload namespace")
	name := fs.String("name", "probe-failure", "Deployment name")
	action := fs.String("action", "rollback", "restart, scale, rollback")
	reason := fs.String("reason", "", "reason for the operation")
	replicas := fs.Int("replicas", -1, "desired replicas for scale")
	planFile := fs.String("plan", "", "reviewed plan JSON file to execute")
	evidence := fs.Bool("evidence", false, "include bounded logs and Prometheus observations")
	timeout := fs.Duration("timeout", 2*time.Minute, "recovery timeout")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	path := "/api/v1/workloads/" + url.PathEscape(*namespace) + "/Deployment/" + url.PathEscape(*name)
	client := http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	call := func(method, path string, body any) ([]byte, error) {
		var data []byte
		var err error
		if body != nil {
			data, err = json.Marshal(body)
			if err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequest(method, strings.TrimRight(*api, "/")+path, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-KubeVista-Operation", "reviewed")
		// Optional cookie for an already authenticated ALB session; never printed.
		if cookie := os.Getenv("KUBEVISTA_SESSION_COOKIE"); cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, b)
		}
		return b, nil
	}
	printJSON := func(b []byte) error {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, b, "", "  "); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, pretty.String())
		return err
	}
	switch command {
	case "diagnose":
		if *evidence {
			path += "?evidence=true"
		}
		data, err := call("GET", path, nil)
		if err != nil {
			return err
		}
		return printJSON(data)
	case "history":
		data, err := call("GET", "/api/v1/operations", nil)
		if err != nil {
			return err
		}
		return printJSON(data)
	case "plan":
		request := cluster.OperationRequest{Action: *action, Namespace: *namespace, Kind: "Deployment", Name: *name, Reason: *reason}
		if *action == "scale" {
			if *replicas < 0 || *replicas > 2147483647 {
				return fmt.Errorf("provide a valid --replicas")
			}
			n := int32(*replicas)
			request.Replicas = &n
		}
		data, err := call("POST", "/api/v1/operations/plan", request)
		if err != nil {
			return err
		}
		var plan cluster.OperationPlan
		if err = json.Unmarshal(data, &plan); err != nil {
			return err
		}
		request.PlanID, request.ExpectedResourceVersion = plan.ID, plan.ResourceVersion
		result, err := json.Marshal(struct {
			Request cluster.OperationRequest `json:"request"`
			Review  cluster.OperationPlan    `json:"review"`
		}{request, plan})
		if err != nil {
			return err
		}
		return printJSON(result)
	case "execute":
		if *planFile == "" {
			return fmt.Errorf("--plan is required; first save and review 'kubevista plan' output")
		}
		data, err := os.ReadFile(*planFile)
		if err != nil {
			return err
		}
		var saved struct {
			Request cluster.OperationRequest `json:"request"`
		}
		if err = json.Unmarshal(data, &saved); err != nil {
			return err
		}
		result, err := call("POST", "/api/v1/operations/execute", saved.Request)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "verify":
		if *timeout <= 0 || *timeout > 10*time.Minute {
			return fmt.Errorf("--timeout must be between 0 and 10m")
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		started := time.Now()
		for {
			data, err := call("GET", path, nil)
			if err != nil {
				return err
			}
			var detail cluster.WorkloadDetail
			if err = json.Unmarshal(data, &detail); err != nil {
				return err
			}
			if detail.Recovery != nil && detail.Recovery.Status == "Recovered" {
				data, _ = json.Marshal(map[string]any{"status": "Recovered", "verificationSeconds": time.Since(started).Seconds(), "recovery": detail.Recovery, "observedAt": detail.ObservedAt})
				return printJSON(data)
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("recovery not verified within %s", *timeout)
			case <-time.After(time.Second):
			}
		}
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}
