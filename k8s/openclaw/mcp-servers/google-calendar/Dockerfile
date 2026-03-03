FROM node:22-slim
WORKDIR /app
COPY package.json ./
RUN npm install --production
COPY index.mjs ./
USER node
EXPOSE 3000
CMD ["node", "index.mjs"]
