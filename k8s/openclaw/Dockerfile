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

USER node
