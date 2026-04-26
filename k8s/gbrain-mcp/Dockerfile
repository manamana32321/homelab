# gbrain-mcp: Streamable HTTP MCP server wrapping garrytan/gbrain over Postgres
# - bun runtime for gbrain CLI
# - mcp-proxy (Python) wraps stdio MCP → SSE/HTTP
# - git CLI for brain repo pull at startup + CronJob sync
FROM oven/bun:1.3.13-alpine

ARG GBRAIN_REF=master

# System deps (git for repo, python+pip for mcp-proxy, ca-certs for HTTPS)
RUN apk add --no-cache \
        git \
        python3 \
        py3-pip \
        ca-certificates \
    && rm -rf /var/cache/apk/*

# Install mcp-proxy (stdio→SSE wrapper)
RUN pip install --break-system-packages --no-cache-dir mcp-proxy

# Install gbrain CLI from upstream
RUN git clone --depth 1 --branch ${GBRAIN_REF} https://github.com/garrytan/gbrain.git /opt/gbrain \
    && cd /opt/gbrain \
    && bun install --production \
    && bun link

# Default working dir holds brain repo (mounted via initContainer or PVC)
WORKDIR /brain

# Default port for mcp-proxy SSE transport
EXPOSE 8080

# mcp-proxy spawns `gbrain serve` (stdio MCP) and exposes it as SSE on :8080.
# Bearer auth handled by caddy sidecar (see manifests/caddy-configmap.yaml).
# --pass-environment: forward GBRAIN_DATABASE_URL etc. to the spawned subprocess
# (mcp-proxy strips env by default, treating stdio servers as untrusted).
# --port/--host: deprecated aliases --sse-port/--sse-host bind to port 0 in
# current mcp-proxy versions; use the canonical flags.
ENTRYPOINT ["mcp-proxy", "--pass-environment", "--port", "8080", "--host", "0.0.0.0", "--", "gbrain", "serve"]
