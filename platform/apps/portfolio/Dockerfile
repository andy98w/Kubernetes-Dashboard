# Build with the portfolio repository as context, not this repository.
FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts
COPY index.html privacy.html styles.css privacy.css vite.config.js ./
COPY src ./src
COPY public ./public
RUN npm run build

FROM ghcr.io/nginx/nginx-unprivileged:1.30.4-alpine
# Provide the server configuration as a separate named build context.
COPY --from=portfolio-config nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
USER 101:101
EXPOSE 8080
