# openfaithmap-web (M2.1, D-AdminSurface) — docker-compose.yml's openfaithmap-web service.
# Standalone Next.js build, no Auth.js/go-oikumenea dependency at all (the anonymous public
# surface never holds a session) — self-contained, no shared workspace with web/apps/admin.

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
COPY --from=build /app/public public
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static .next/static
EXPOSE 3000
CMD ["node", "server.js"]
