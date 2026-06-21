# syntax=docker/dockerfile:1.6

# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install only what the build needs; no dev tooling persists to the final image.
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /build

# Layer dependencies separately to exploit the build cache.
COPY go.mod go.sum ./
RUN go mod download -x

COPY . .

# CGO_ENABLED=0: pure Go binary, no libc dependency — required for distroless.
# -trimpath: strip local paths from stack traces (security + reproducibility).
# -ldflags -w -s: drop DWARF and symbol table to reduce binary size.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-w -s" \
      -o /build/ms_home \
      ./cmd/home

# ── Final image ───────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS final

# Bring timezone data and CA certs from the builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /build/ms_home /ms_home

# nonroot image already runs as uid 65532 — no USER directive needed.
# Read-only root filesystem; Cloud Run sets PORT automatically.
EXPOSE 8080

ENTRYPOINT ["/ms_home"]
