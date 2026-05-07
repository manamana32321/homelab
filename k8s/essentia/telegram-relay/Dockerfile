FROM node:22-alpine

WORKDIR /app
COPY relay.mjs ./

USER node
EXPOSE 8080
CMD ["node", "relay.mjs"]
