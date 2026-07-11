# Spec: DevOps & DX Tooling

## Objective
Improve the Developer Experience (DX) for contributors by setting up hot-reloading (`air`), pre-commit hooks (`gofmt`, `golangci-lint`), a Devcontainer/Codespaces configuration for instant cloud environments, and one-click deploy buttons for platforms like Railway or Render.

## Tech Stack
- Go (`air` for hot reload, `golangci-lint`)
- Git hooks (`pre-commit`)
- Docker (`.devcontainer`)

## Commands
Setup: `make dev-setup` (installs air and hooks)
Dev: `make dev` (starts the stack with air)
Lint: `make lint`

## Project Structure
```
.devcontainer/
  devcontainer.json       → Configuration for VS Code / Codespaces
  Dockerfile              → Dev environment container definition
.air.toml                 → Config file for the Air hot-reloader
.pre-commit-config.yaml   → Pre-commit framework config
Makefile                  → Update with new `dev` and `lint` targets
```

## Code Style
```yaml
# .air.toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/main ./cmd/server"
bin = "./tmp/main"
include_ext = ["go", "tpl", "tmpl", "html", "env"]
exclude_dir = ["assets", "tmp", "vendor", "frontend"]
delay = 1000 # ms
```

## Testing Strategy
- **Manual Verification**: Run `make dev`, make a change to a Go file, and verify the server automatically recompiles and restarts within seconds.
- **Codespaces Check**: Open the repository in a GitHub Codespace and verify the Go extension, Node.js, and PostgreSQL are all pre-configured and ready to use.

## Boundaries
- **Always**: Ensure that `make dev` works seamlessly on macOS, Linux, and WSL2.
- **Ask first**: Before adding heavyweight global tools to the `dev-setup` script. If a tool isn't strictly necessary for a basic contribution, don't force its installation.
- **Never**: Commit local OS-specific binaries or `.env` files. Ensure the `.gitignore` covers `tmp/` and `bin/`.

## Success Criteria
- [ ] A new contributor can clone the repo and run `make dev` to get a hot-reloading backend environment.
- [ ] A developer can click "Open in Codespaces" on GitHub and get a fully functioning, containerized IDE in the browser.
- [ ] A `Deploy to Railway` button is added to the `README.md` that successfully provisions the PostgreSQL DB and deploys the Go API.

## Open Questions
- Do we want to enforce `pre-commit` globally for all contributors, or just recommend it and rely on CI to catch linting errors?
