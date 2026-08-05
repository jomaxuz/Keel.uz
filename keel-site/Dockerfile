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
# This project has no static assets of its own — the logo is inline SVG and the
# fonts come from next/font. Created rather than conditionally copied because
# `COPY` has no "if it exists", and an empty directory is cheaper than a
# Dockerfile that breaks the day somebody adds a favicon.
RUN mkdir -p public
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- runner ----
FROM node:22-alpine AS runner
# The landing page and the console both render dates server-side; without
# tzdata the runner silently ignores TZ and renders in UTC.
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
EXPOSE 3100
ENV PORT=3100
CMD ["node", "server.js"]
