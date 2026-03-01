# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /build

# Copy package files
COPY package*.json ./

# Install all dependencies (devDependencies needed for vite build)
RUN npm ci

# Copy source
COPY . .

# Build for production
RUN npm run build

# Stage 2: Runtime with nginx
FROM nginx:1.25-alpine

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder
COPY --from=builder /build/dist /usr/share/nginx/html

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80 || exit 1

# Expose port
EXPOSE 80

# Start nginx (master process runs as root to bind port 80, workers run as nginx user)
CMD ["nginx", "-g", "daemon off;"]
