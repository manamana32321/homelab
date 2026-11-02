FROM alpine:3.20 AS font
RUN apk add --no-cache font-noto-cjk py3-fonttools
RUN pyftsubset /usr/share/fonts/noto/NotoSansCJK-Regular.ttc --font-number=1 \
      --unicodes="U+0020-007E,U+00A0-00FF,U+2010-2027,U+2190-21FF,U+2212,U+3000-303F,U+3131-318E,U+AC00-D7A3,U+FF01-FF5E" \
      --layout-features="*" --output-file=/NotoSansKR-subset.otf

FROM node:22-alpine

COPY --from=font /NotoSansKR-subset.otf /usr/share/fonts/chart/NotoSansKR-subset.otf

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund
COPY relay.mjs format.mjs chart.mjs ./

USER node
EXPOSE 8080
CMD ["node", "relay.mjs"]
