package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LogExcerpt struct {
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Previous  bool   `json:"previous"`
	Text      string `json:"text"`
}
type MetricSample struct {
	Name  string    `json:"name"`
	Value string    `json:"value"`
	At    time.Time `json:"at"`
}

// Enrich is opt-in so opening a workload does not automatically retrieve logs.
func (c *Client) Enrich(ctx context.Context, detail *WorkloadDetail) {
	defer func() {
		sort.SliceStable(detail.Timeline, func(i, j int) bool { return detail.Timeline[i].At.Before(detail.Timeline[j].At) })
	}()
	for _, p := range detail.Pods {
		if len(detail.Logs) >= 3 {
			break
		}
		if p.Ready == p.Containers && p.Restarts == 0 {
			continue
		}
		pod, e := c.client.CoreV1().Pods(detail.Workload.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
		if e != nil {
			detail.Warnings = append(detail.Warnings, "Pod logs unavailable: "+p.Name)
			continue
		}
		statuses := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for _, s := range statuses {
			if len(detail.Logs) >= 3 {
				break
			}
			if s.Ready && s.RestartCount == 0 {
				continue
			}
			previous := s.RestartCount > 0
			lines, bytes := int64(30), int64(8192)
			sub, cancel := context.WithTimeout(ctx, time.Second)
			data, err := c.client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: s.Name, Previous: previous, TailLines: &lines, LimitBytes: &bytes, Timestamps: true}).DoRaw(sub)
			cancel()
			if err != nil {
				detail.Warnings = append(detail.Warnings, "Logs unavailable: "+pod.Name+"/"+s.Name)
				continue
			}
			detail.Logs = append(detail.Logs, LogExcerpt{pod.Name, s.Name, previous, string(data)})
		}
	}
	base := os.Getenv("KUBEVISTA_PROMETHEUS_URL")
	if base == "" {
		detail.Warnings = append(detail.Warnings, "Prometheus is not configured; no application metrics were queried")
		return
	}
	ns, name := strconv.Quote(detail.Workload.Namespace), strconv.Quote(detail.Workload.Name)
	// These conventional application metrics must be labeled at scrape time.
	queries := []struct{ name, query string }{
		{"HTTP error ratio (5m)", "sum(rate(http_requests_total{namespace=" + ns + ",deployment=" + name + ",status=~\"5..\"}[5m])) / sum(rate(http_requests_total{namespace=" + ns + ",deployment=" + name + "}[5m]))"},
		{"HTTP p99 seconds (5m)", "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{namespace=" + ns + ",deployment=" + name + "}[5m])))"},
	}
	for _, q := range queries {
		sample, err := queryMetric(ctx, base, q.name, q.query)
		if err != nil {
			detail.Warnings = append(detail.Warnings, q.name+": unavailable (check metric names and namespace/deployment labels)")
			continue
		}
		detail.Metrics = append(detail.Metrics, sample)
		detail.Timeline = append(detail.Timeline, IncidentEvidence{"Prometheus", sample.Name + ": " + sample.Value + "; observation, not proof of causation", sample.At})
	}
}

func queryMetric(ctx context.Context, base, name, query string) (MetricSample, error) {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return MetricSample{}, fmt.Errorf("invalid Prometheus URL")
	}
	u.Path = "/api/v1/query"
	u.RawQuery = url.Values{"query": []string{query}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return MetricSample{}, err
	}
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return MetricSample{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return MetricSample{}, fmt.Errorf("Prometheus returned %d", resp.StatusCode)
	}
	var payload struct {
		Status string
		Data   struct {
			Result []struct{ Value []json.RawMessage }
		}
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&payload); err != nil {
		return MetricSample{}, err
	}
	if payload.Status != "success" || len(payload.Data.Result) != 1 || len(payload.Data.Result[0].Value) != 2 {
		return MetricSample{}, fmt.Errorf("no unique metric sample")
	}
	var stamp float64
	var value string
	if json.Unmarshal(payload.Data.Result[0].Value[0], &stamp) != nil || json.Unmarshal(payload.Data.Result[0].Value[1], &value) != nil || value == "NaN" || value == "+Inf" {
		return MetricSample{}, fmt.Errorf("invalid sample")
	}
	return MetricSample{name, value, time.UnixMilli(int64(stamp * 1000)).UTC()}, nil
}
