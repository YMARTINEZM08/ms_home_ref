# Deployment

## Prerequisites

- GCP project with Cloud Run API enabled
- Artifact Registry repository for Docker images
- Service account `ms-home-liverpool-sa` with roles:
  - `roles/run.invoker` (for health checks)
  - `roles/secretmanager.secretAccessor` (for `content-service-url` secret)
- Secret `content-service-url` in Secret Manager

## Build and push

```bash
IMAGE=REGION-docker.pkg.dev/PROJECT_ID/REPO/ms-home-liverpool

docker build -f deployments/Dockerfile -t $IMAGE:$TAG .
docker push $IMAGE:$TAG
```

## Deploy to Cloud Run

```bash
gcloud run services replace deployments/service.yaml \
  --region REGION \
  --project PROJECT_ID
```

Replace `PROJECT_ID`, `REGION`, `REPO`, and `TAG` with environment-specific values.

## Environment variables

See [configs/.env.example](../configs/.env.example) for the full list with descriptions.

Required at runtime:
- `CONTENT_SERVICE_URL` — injected from Secret Manager in `service.yaml`

Optional / with defaults:
- `PORT` (default: `8080`)
- `LOG_LEVEL` (default: `INFO`)
- `BREAKER_FAILURE_RATIO` (default: `0.05`)
- `BREAKER_MIN_REQUESTS` (default: `20`)
- `BREAKER_OPEN_TIMEOUT_S` (default: `30`)
- `OTEL_SAMPLE_RATIO` (default: `1.0`)

## Health probes

| Path | Type | Notes |
|---|---|---|
| `/healthz` | Liveness | Returns 200 immediately if the process is running |
| `/readyz` | Readiness | Returns 200 if config is valid; Cloud Run stops routing until ready |

## Graceful shutdown

The server listens for `SIGTERM`/`SIGINT` (Cloud Run sends `SIGTERM` before scaling down). It calls `http.Server.Shutdown` with a 30-second context, then shuts down the OTel provider. In-flight requests have up to 30 seconds to complete.

## Local development

```bash
cp configs/.env.example configs/.env.local
# Edit .env.local with real values

export $(grep -v '^#' configs/.env.local | xargs)
go run ./cmd/home

# Verify
curl -s localhost:8080/healthz | jq
curl -s localhost:8080/home -H 'x-brand-id: LP' | jq
```
