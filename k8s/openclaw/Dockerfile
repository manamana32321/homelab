FROM ghcr.io/openclaw/openclaw:latest

USER root

RUN apt-get update && apt-get install -y --no-install-recommends \
  libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 \
  libxkbcommon0 libxcomposite1 libxdamage1 libxrandr2 libgbm1 \
  libpango-1.0-0 libcairo2 libasound2 libxshmfence1 \
  libx11-6 libxcb1 libxext6 libxfixes3 libatspi2.0-0 \
  libdbus-1-3 libglib2.0-0 \
  fonts-liberation \
  && rm -rf /var/lib/apt/lists/*

# gws CLI — musl static binary to avoid glibc version mismatch (bookworm = 2.36, gnu build requires 2.39)
ARG GWS_VERSION=0.22.5
RUN curl -fsSL "https://github.com/googleworkspace/cli/releases/download/v${GWS_VERSION}/google-workspace-cli-x86_64-unknown-linux-musl.tar.gz" \
  | tar xz --strip-components=0 -C /usr/local/bin ./gws \
  && chmod +x /usr/local/bin/gws

# kubectl for Jarvis K8s readonly access (#84)
RUN curl -fsSL "https://dl.k8s.io/release/$(curl -fsSL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
  -o /usr/local/bin/kubectl \
  && chmod +x /usr/local/bin/kubectl

USER node
