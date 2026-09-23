# Read-only investigation prototype

This local-only API experiment builds a bounded evidence packet from the existing
workload investigation path, then optionally asks a local Ollama model to explain
it. It does not execute operations, generate PromQL/LogQL, or query Loki directly.
Logs currently come from the Kubernetes container-log API; Prometheus samples
come from the existing optional enrichment path. No cloud AI account is needed.

## Try the evidence preview

From `api/`, run:

```sh
KUBEVISTA_ADDRESS=127.0.0.1:8080 KUBEVISTA_DEMO_MODE=true \
KUBEVISTA_INVESTIGATION_ENABLED=true go run ./cmd/server
```

In another terminal:

```sh
curl -sS -X POST \
  -H 'X-KubeVista-Investigation: reviewed' \
  http://127.0.0.1:8080/api/v1/investigate/kubevista/Deployment/kubevista-api
```

Without a model, the response explicitly says `evidence-only`. Demo evidence is
also labeled. To use live data, disable demo mode and select a local kubeconfig;
the allowlist uses `KUBEVISTA_OPERATION_NAMESPACES` but does not enable writes.

## Enable local inference

Run Ollama locally with a downloaded **local** model, then set
`KUBEVISTA_LOCAL_MODEL` to its exact installed name when starting the API.
Use a local model, not an Ollama cloud-backed model, for private evidence.
The endpoint is fixed to `127.0.0.1:11434/api/chat`; redirects and HTTP proxies
are disabled. No model was downloaded or inference service installed by this change.

The response contains up to three hypotheses, each citing packet IDs. Look up
each ID in `evidence`. Citation validation rejects unknown IDs; it does **not**
prove that a hypothesis is true or that a citation supports it. Always review.

## Boundaries and limitations

- Disabled by default and always disabled when environment is `production`.
- Bind the prototype API to loopback; it is not a public authenticated service.
- Explicit POST header, namespace allowlist, one concurrent investigation per process.
- Five-second collection budget, six-second inference budget, twelve seconds overall.
- At most 24 evidence items, three log excerpts, 1,500 bytes per excerpt/item.
- Structured JSON output, response-size cap, no tools or mutation access for the model.
- Common credential patterns are redacted before inference. Redaction is best-effort:
  arbitrary secrets, personal data, and multiline credentials may remain. Only use
  synthetic/sanitized workloads until a stronger data policy has been evaluated.
- Log and event text is explicitly treated as untrusted input. Prompt injection
  can still influence prose; model output must never authorize an operation.
- No persistence, background jobs, hosted provider, UI integration, or automated fixes.
- Slow/cold model loads, invalid JSON, or invalid citations fall back to evidence only.

Tests use HTTP model fixtures, not a real model. These establish access boundaries,
redaction/bounds, and citation validation—not diagnosis quality. Before promoting
this experiment, evaluate real model answers against the five incident-lab cases,
healthy/recovered workloads, missing telemetry, and prompt-injection examples.
