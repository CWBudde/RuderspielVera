# Ruderspiel server image: Go API + built SPA. Build: `just image`; deploy: `just deploy`.

FROM node:24-alpine AS web
WORKDIR /src/web
RUN corepack enable && corepack prepare pnpm@10.29.2 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.27.4-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ruderspiel ./cmd/server && mkdir -p /out/data/games

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/ruderspiel /ruderspiel
COPY --from=web /src/web/dist /web
# Owned by the nonroot user, so a fresh Docker volume on /data is writable too.
COPY --from=server --chown=65532:65532 /out/data /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/ruderspiel", "-addr", ":8080", "-data", "/data/games", "-web", "/web"]
