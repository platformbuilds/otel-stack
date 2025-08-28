# --- Build stage --------------------------------------------------------------
FROM node:20-alpine AS build
WORKDIR /app

# Install deps with cache-friendly copy
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./
RUN \
  if [ -f package-lock.json ]; then npm ci --no-audit --no-fund; \
  elif [ -f pnpm-lock.yaml ]; then corepack enable && pnpm i --frozen-lockfile; \
  elif [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
  else npm i --no-audit --no-fund; fi

# Copy source and build
COPY . .
# If you ever want compile-time API base, uncomment:
# ARG VITE_API_BASE
# ENV VITE_API_BASE=$VITE_API_BASE
RUN npm run build

# --- Runtime stage ------------------------------------------------------------
FROM caddy:2-alpine
# Static bundle
COPY --from=build /app/dist /srv
# Caddy config
COPY Caddyfile /etc/caddy/Caddyfile

# Default backend; override at runtime: -e API_BASE=http://host.docker.internal:8080
ENV API_BASE=http://backend:8080

EXPOSE 80
# caddy image entrypoint runs: caddy run --config /etc/caddy/Caddyfile --adapter caddyfile