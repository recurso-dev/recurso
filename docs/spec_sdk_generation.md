# Spec: SDK Generation & API Playground

> **Superseded (2026-07-16):** the SDKs no longer live under `sdk/` in this
> repo. They were extracted to standalone repositories:
> [recurso-go](https://github.com/recurso-dev/recurso-go),
> [recurso-node](https://github.com/recurso-dev/recurso-node),
> [recurso-python](https://github.com/recurso-dev/recurso-python).
> Paths below reflect the old monorepo layout and are kept for historical
> context.

## Objective
Automate the generation of client SDKs (Python and Go) directly from the OpenAPI specification, wire up a Mintlify-backed API playground for interactive documentation, and provide an auto-generated Postman collection for developers.

## Tech Stack
- OpenAPI 3.1 (existing `openapi.json`)
- `openapi-generator-cli` or `Speakeasy` (for SDK generation)
- Mintlify (for documentation/playground)
- Postman API (for collection generation)

## Commands
Generate SDKs: `make generate-sdks`
Preview Docs: `npx mintlify dev`

## Project Structure
```
sdk/
  python/                 → Auto-generated Python client
  go/                     → Auto-generated Go client
docs/
  mint.json               → Mintlify configuration
  api-reference/          → Mintlify MDX files pointing to openapi.json
scripts/
  generate-sdks.sh        → Bash script to run the openapi generator
```

## Code Style
```bash
#!/bin/bash
# scripts/generate-sdks.sh

# Generate Python SDK
openapi-generator-cli generate -i ./openapi.json -g python -o ./sdk/python --package-name recurso

# Generate Go SDK
openapi-generator-cli generate -i ./openapi.json -g go -o ./sdk/go --package-name recurso
```

## Testing Strategy
- **SDK Compilation**: The CI pipeline must compile the generated Go SDK (`go build ./sdk/go/...`) and the Python SDK to ensure the generator didn't produce syntactically invalid code.
- **Playground Verification**: Run the Mintlify dev server and manually verify that clicking "Test Endpoint" works successfully against a local instance.

## Boundaries
- **Always**: Ensure the generation script can be run locally without requiring paid cloud accounts (e.g., if using a cloud generator, it must have a free local CLI mode).
- **Ask first**: Before committing generated code directly to the main repository if it makes the PRs massive. Consider publishing them to separate repositories (e.g., `recurso-dev/recurso-python`) via GitHub Actions.
- **Never**: Manually edit the generated files in `sdk/python` or `sdk/go`. All changes must happen via the `openapi.json` spec or generator templates.

## Success Criteria
- [ ] Running `make generate-sdks` successfully creates `sdk/python` and `sdk/go`.
- [ ] The Mintlify documentation site has an interactive API playground powered by the OpenAPI spec.
- [ ] A Postman collection is linked in the README, allowing users to import all endpoints instantly.

## Open Questions
- Do we want to use the open-source `openapi-generator-cli`, or a commercial tool like `Speakeasy` which produces higher-quality, idiomatic SDKs?
- Should the generated SDKs live in this mono-repo, or should CI push them to their own respective GitHub repositories?
