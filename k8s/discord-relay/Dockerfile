FROM node:22-alpine

RUN apk add --no-cache font-dejavu

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --no-audit --no-fund
COPY relay.mjs format.mjs chart.mjs ./

USER node
EXPOSE 8080
CMD ["node", "relay.mjs"]
