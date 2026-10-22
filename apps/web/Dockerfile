FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
# Stamped into the image so the deploy can prove which commit is actually serving.
ARG FT_VERSION=dev
ARG FT_COMMIT=unknown
RUN mkdir -p public && printf '{"version":"%s","commit":"%s"}\n' "$FT_VERSION" "$FT_COMMIT" > public/version.json
RUN pnpm build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3000 HOSTNAME=0.0.0.0
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
