# Read-only investigation prototype

This local-only API experiment builds a bounded evidence packet from the existing
workload investigation path, then optionally asks a local Ollama model to explain
it. It does not execute operations, generate PromQL/LogQL, or query Loki directly.
Logs currently come from the Kubernetes container-log API; Prometheus samples
come from the existing optional enrichment path. No cloud AI account is needed.

## Workload inspector

Open a robot, then choose **Investigate workload** in its diagnosis section.
The result shows an observation time, hypotheses when a local model is available,
and evidence IDs that focus the corresponding source excerpt. Model prose is
rendered as text, not HTML or executable commands. There are no action buttons
in model answers. Existing guarded operations remain a separate human workflow.

In the static browser demo this is explicitly **Demo evidence · no AI model**.
It summarizes the selected synthetic incident without calling the API or an LLM.
An API-backed UI calls the existing POST endpoint. Disabled, disallowed, busy,
and failed investigations show errors rather than invented explanations. Refreshing
the observation or switching workloads clears the old answer; requests cancel on
unmount, and users can cancel explicitly.

Logs and metric enrichment require the **Include bounded logs and metrics**
checkbox, which sends `X-KubeVista-Include-Logs: true`. They are excluded by default.
Only use this option for synthetic or sanitized workloads. The allowlist is a
server-wide prototype restriction, not per-user namespace authorization. The
production disable remains in place until authenticated per-user access and a
data-handling policy are implemented and verified.

## Try the evidence preview

First set `KUBEVISTA_INVESTIGATION_TOKEN` to a randomly generated value of at least
32 characters (for example, `export KUBEVISTA_INVESTIGATION_TOKEN="$(openssl rand -hex 32)"`).
Enter it in the live investigation inspector; it is not stored in the browser.
The shared credential is a local access gate, not production per-user authorization.

From `api/`, run:

```sh
KUBEVISTA_ADDRESS=127.0.0.1:8080 KUBEVISTA_DEMO_MODE=true \
KUBEVISTA_INVESTIGATION_ENABLED=true \
KUBEVISTA_INVESTIGATION_TOKEN="$KUBEVISTA_INVESTIGATION_TOKEN" go run ./cmd/server
```

In another terminal:

```sh
curl -sS -X POST \
  -H 'X-KubeVista-Investigation: reviewed' \
  -H "Authorization: Bearer $KUBEVISTA_INVESTIGATION_TOKEN" \
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
- No persistence, background jobs, hosted provider, or automated fixes.
- Slow/cold model loads, invalid JSON, or invalid citations fall back to evidence only.

Tests use HTTP model fixtures, not a real model. These establish access boundaries,
redaction/bounds, and citation validation—not diagnosis quality. Before promoting
this experiment, evaluate real model answers against the five incident-lab cases,
healthy/recovered workloads, missing telemetry, and prompt-injection examples.
