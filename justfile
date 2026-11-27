# Ruderspiel dev tasks

default:
    @just --list

test:
    go test ./...

vet:
    go vet ./...

server:
    go run ./cmd/server -addr :8080

web:
    cd web && pnpm dev --host

# server + vite dev (vite proxies /api to :8080)
dev:
    #!/usr/bin/env bash
    trap 'kill 0' EXIT
    go run ./cmd/server -addr :8080 &
    cd web && pnpm dev --host

build:
    cd web && pnpm install --frozen-lockfile && pnpm build
    go build -o bin/ruderspiel ./cmd/server

sim games="500" seed="1":
    go run ./cmd/sim -games {{games}} -seed {{seed}}

gendata:
    go run ./cmd/gendata -out internal/data/rowers.json

gen-types:
    go run github.com/gzuidhof/tygo@latest generate
