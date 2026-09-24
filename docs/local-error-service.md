# Local error service

The fleet inspector has two separate workflows. The example buttons exercise
in-memory grouping; those groups reset on reload. **Connected error service**
captures real browser error categories and saves them through the Go API.

## Run locally

Generate a random credential in your terminal and keep it out of Git:

```sh
export KUBEVISTA_ISSUE_TOKEN="$(openssl rand -hex 32)"
export KUBEVISTA_ISSUE_STORE=/private/tmp/kubevista-local/issues.json
export KUBEVISTA_ADDRESS=127.0.0.1:8088
export KUBEVISTA_DEMO_MODE=true
cd api
go run ./cmd/server
```

Start the web app in a second terminal with `VITE_API_TARGET=http://127.0.0.1:8088`
and `VITE_DATA_MODE=demo`. Open `?fleet=1#/station`, select the error desk,
and enter the credential into Connected error service. It stays in React memory,
not local storage or the web bundle. Closing the inspector or disconnecting stops
capture. Reconnect to retrieve saved issues.

**Trigger browser error** throws a controlled TypeError. Resolve the saved issue,
then trigger again: the same group becomes Regressed. The API stores its count,
first/last timestamps, release, and version. Resolution uses compare-and-swap;
an event arriving after your read causes a 409 rather than silently resolving it.

## Storage and access

- Every endpoint requires a bearer credential of at least 32 characters.
- Requests are limited to 2 KiB and four constrained identifiers. Unknown fields
  and URL-shaped values are rejected. The browser sends no exception message,
  stack trace, request body, cookies, or user URL.
- Ingestion is capped at 60 events/minute per process and 1,000 groups. A full
  store rejects new groups; it does not silently evict unresolved issues.
- Writes use a private temporary file, sync, then atomic rename. Acknowledgment
  follows persistence. A corrupt existing store fails closed.
- Use one process per file. There is no distributed lock, shared database,
  directory fsync guarantee, retention job, or HA support. A `/private/tmp` file
  survives process restarts but can be removed by the OS; use a private durable
  directory if you need longer retention.
- This service is disabled when the API environment is `production`. Bind it to
  loopback. The shared local credential is not per-user authorization.

This is not a Sentry replacement. Source maps, stack capture with reviewed
redaction, trace correlation, alerts, ownership, retention, and production
identity/storage still need separate implementation and validation. Keep Sentry
in existing applications while evaluating those capabilities.
