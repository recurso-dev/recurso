# Contributing to Recurso

Thanks for your interest in contributing! This guide covers how to get a
development environment running and what we expect from changes.

## Development setup

Prerequisites: Go 1.23+, Docker with Compose, `jq` (used by the e2e scripts),
and Node 18+ if you're working on the frontend or docs site.

```bash
# Start Postgres, TigerBeetle, and Mailhog (plus the API in a container)
make docker-up

# Or run the API locally against the containerized dependencies
make run
```

The API listens on `http://localhost:8080`. The dashboard lives in
`frontend/` (`npm install && npm run dev`); the Vite dev server proxies API
requests, so no extra configuration is needed.

## Testing

```bash
make test         # Go unit tests
make test-e2e     # End-to-end flow against a running stack
make test-verify  # Phase verification scripts (scripts/verify/)
cd frontend && npx vitest run   # Frontend tests
```

CI runs lint (`golangci-lint`), unit tests, and the e2e suite on every pull
request — please make sure `make lint`, `make test`, and `make test-e2e` pass
locally before opening one.

## Making changes

- Keep changes focused; separate unrelated fixes into separate PRs.
- New backend behavior needs a migration when it touches the schema
  (`internal/adapter/db/migrations/`, sequential numbering, with a matching
  `.down.sql`).
- Every query on tenant-owned data must filter by `tenant_id`. If you add a
  cross-tenant lookup (e.g. for webhook routing), document why on the method.
- Add tests for anything that computes money, tax, or proration.

## Reporting issues

Open a GitHub issue with reproduction steps, expected vs actual behavior, and
relevant logs. For suspected security vulnerabilities, please do not open a
public issue — email the maintainers instead.

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).
