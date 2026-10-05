# Build stage: use pnpm from repo root (build context must be repo root)
FROM node:26-alpine AS builder

WORKDIR /app

# Install the pnpm version declared in package.json. A global pnpm 11 install
# tries to launch a newer packageManager build as a Node module and crashes
# when that build is a native binary.
COPY package.json ./
RUN npm install -g "$(node -e 'process.stdout.write(require("./package.json").packageManager)')"

# Copy workspace and frontend package files (when context is repo root: docker build -f frontend/Dockerfile .)
COPY pnpm-lock.yaml pnpm-workspace.yaml ./
COPY frontend/package.json frontend/package.json

# Install dependencies (workspace root)
RUN pnpm install --frozen-lockfile

# Copy frontend source
COPY frontend frontend

WORKDIR /app/frontend

# Build
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
RUN pnpm run build

# Production stage
FROM nginx:stable-alpine

# Upgrade openssl packages for libssl3 CVEs reported by Trivy
RUN apk --no-cache upgrade libssl3 libcrypto3 openssl

# Copy built assets
COPY --from=builder /app/frontend/dist /usr/share/nginx/html

# Copy nginx config
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
