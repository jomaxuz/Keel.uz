# syntax=docker/dockerfile:1

# ---- deps ----
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install

# ---- builder ----
FROM node:22-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* values are inlined into the client bundle at build time, so they
# must be present here (docker-compose passes them as build args).
ARG NEXT_PUBLIC_MAP_API_KEY=""
ENV NEXT_PUBLIC_MAP_API_KEY=$NEXT_PUBLIC_MAP_API_KEY
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- runner ----
FROM node:22-alpine AS runner
# Server-rendered pages format dates with the container's local time, so the
# runner needs the timezone database too — without it TZ is silently ignored
# and SSR renders in UTC while the browser renders in Tashkent.
RUN apk add --no-cache tzdata
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup -g 1001 -S nodejs && adduser -S nextjs -u 1001

# Next.js standalone output (see next.config.ts `output: "standalone"`).
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000
CMD ["node", "server.js"]
