# Ruderspiel dev tasks

# Prefer tools installed by `just setup-deps` (GOPATH/bin) over older system copies.
export PATH := `go env GOPATH` + "/bin:" + env_var("PATH")

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

#################################
# Formatting & linting
#################################

# Install pinned formatters and linters (versions follow MeKo-Tech/ewws-render)
setup-deps:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Installing development dependencies..."
    gobin="$(go env GOPATH)/bin"
    mkdir -p "$gobin"

    # treefmt: `go install` fails on the module zip (emoji in a test path), so fetch the release binary.
    "$gobin/treefmt" --version 2>/dev/null | grep -qF 'v2.6.0' || { echo "Installing treefmt v2.6.0..."; curl -fsSL https://github.com/numtide/treefmt/releases/download/v2.6.0/treefmt_2.6.0_linux_amd64.tar.gz | tar -C "$gobin" -xz treefmt; }

    # golangci-lint: 2.12.x cannot type-check under Go 1.27.
    "$gobin/golangci-lint" version 2>/dev/null | grep -qF 'version 2.15.0 ' || { echo "Installing golangci-lint v2.15.0..."; go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0; }

    "$gobin/gofumpt" --version 2>/dev/null | grep -qF 'v0.12.0' || { echo "Installing gofumpt v0.12.0..."; go install mvdan.cc/gofumpt@v0.12.0; }
    "$gobin/gci" --version 2>/dev/null | grep -qF '0.14.0' || { echo "Installing gci v0.14.0..."; go install github.com/daixiang0/gci@v0.14.0; }
    "$gobin/shfmt" --version 2>/dev/null | grep -qF 'v3.14.1' || { echo "Installing shfmt v3.14.1..."; go install mvdan.cc/sh/v3/cmd/shfmt@v3.14.1; }

    prettier --version 2>/dev/null | grep -qF '3.9.10' || { echo "Installing prettier 3.9.10..."; npm install -g prettier@3.9.10 || echo "prettier installation failed - install Node.js/npm first."; }

# Format all files (Go, web, shell)
fmt:
    treefmt --allow-missing-formatter

# Fail if formatting would change anything (note: still rewrites files in place)
fmt-check:
    treefmt --allow-missing-formatter --fail-on-change

# Run golangci-lint
lint:
    golangci-lint run --config ./.golangci.toml --timeout 5m ./...

# Run golangci-lint and apply auto-fixes
lint-fix:
    golangci-lint run --config ./.golangci.toml --timeout 5m --fix ./...
