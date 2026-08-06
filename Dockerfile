# Multi-stage build for the openfaithmap-api server. pgx is pure Go (once wired), so the binary is
# built CGO-free and shipped on distroless/static (no libc, non-root). Migrations are NOT run by
# this image — apply them out-of-band (atlas) before starting the server. See docker-compose.yml.
# Mirrors go-oikumenea's own Dockerfile shape (docs/architecture/decisions.md#d-stack).

# ---- build ----
# --platform=$BUILDPLATFORM pins the build stage to the BUILDER's architecture and cross-compiles
# from there instead of running an emulated toolchain (see go-oikumenea's Dockerfile for the full
# rationale — pure-Go + CGO off makes cross-compiling free).
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
# Module graph first for layer caching.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/openfaithmap-api ./cmd/openfaithmap-api

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
# Default config baked in; operators override by mounting their own var/conf.
COPY --from=build /src/var/conf /app/var/conf
COPY --from=build /out/openfaithmap-api /app/openfaithmap-api
# 3000 = app API, 3001 = management (health/readiness/debug).
EXPOSE 3000 3001
USER nonroot:nonroot
ENTRYPOINT ["/app/openfaithmap-api"]
CMD ["serve"]
