#!/usr/bin/env python3
"""Generate llms.txt and llms-full.txt from the OpenAPI spec.

These are machine-readable files served at the root of recurso.dev so that AI
coding assistants and agents can integrate the Recurso REST API correctly
without crawling the site:

  - llms.txt       concise index (the llmstxt.org convention): positioning,
                   conventions, curated links, and a by-resource endpoint map.
  - llms-full.txt  the complete reference: every endpoint plus every request /
                   response object as a real JSON Schema.

The source of truth is cmd/api/openapi.yaml, which CI drift-gates against the
registered routes — so regenerating here keeps the LLM files honest.

Usage:
    python3 scripts/gen_llms.py [--spec cmd/api/openapi.yaml] [--out DIR]

DIR defaults to the current directory; point it at the website's public/ dir.
"""
import argparse
import json
import os
import subprocess
import sys

# Curated positioning + links. Kept in code (not derived from the spec) so the
# marketing voice stays deliberate and matches the site hero.
TAGLINE = (
    "Recurso is the AI-native billing platform for developer-first products: "
    "metering, subscriptions, and tax-correct invoicing on a real double-entry "
    "ledger. Open-source and self-hostable, with US sales tax, EU e-invoicing, "
    "and India GST built in — and no cut of your revenue."
)

LINKS = [
    ("Quickstart", "https://docs.recurso.dev/quickstart",
     "Create a plan, a customer, and your first subscription in minutes."),
    ("Core concepts", "https://docs.recurso.dev/concepts",
     "The data model: plans, subscriptions, invoices, the ledger, and money as minor units."),
    ("API reference", "https://docs.recurso.dev/api-reference/introduction",
     "Human-readable reference with request/response examples for every endpoint."),
    ("Full machine reference", "https://recurso.dev/llms-full.txt",
     "Every endpoint and every object as a JSON Schema — the companion to this file."),
    ("MCP server", "https://docs.recurso.dev/advanced/mcp-server",
     "Operate billing from an AI agent over the Model Context Protocol."),
    ("SDKs", "https://docs.recurso.dev/sdk",
     "Official Node, Python, and Go clients."),
    ("Live sandbox", "https://demo.recurso.dev/",
     "A seeded, resettable dashboard — no signup."),
]

SDKS = [
    ("Node / TypeScript", "https://github.com/recurso-dev/recurso-node"),
    ("Python", "https://github.com/recurso-dev/recurso-python"),
    ("Go", "https://github.com/recurso-dev/recurso-go"),
]

CONVENTIONS = """\
- **Base URL**: your own deployment (the hosted cloud is `https://api.recurso.dev`). Substitute it for the example URL in the spec.
- **Auth**: `Authorization: Bearer <api_key>`. Obtain a key by registering a tenant via `POST /auth/register`.
- **Versioning**: all authenticated endpoints live under the `/v1` prefix.
- **Money**: integer **minor units** (`int64`) — cents, paise. `2000` in a 2-decimal currency is 20.00. Respect the currency's exponent (JPY has 0, KWD/BHD have 3).
- **Idempotency**: mutating requests accept an `X-Idempotency-Key` header; retries with the same key are safe.
- **Errors**: non-2xx responses return a JSON body with a machine-readable `error` code and a human `message`.
- **Pagination**: list endpoints take `limit` and `offset`; some default to a small limit, so pass an explicit `limit` when you need the full set."""


def load_spec(path):
    """Load the OpenAPI YAML. Uses PyYAML if present, else shells to Ruby
    (which ships with macOS) to convert YAML -> JSON."""
    try:
        import yaml  # type: ignore
        with open(path) as f:
            return yaml.safe_load(f)
    except ModuleNotFoundError:
        pass
    try:
        out = subprocess.check_output(
            ["ruby", "-ryaml", "-rjson", "-e",
             "puts YAML.safe_load(File.read(ARGV[0]), aliases: true).to_json", path]
        )
        return json.loads(out)
    except (OSError, subprocess.CalledProcessError) as e:
        sys.exit(f"Cannot parse YAML: install PyYAML (pip install pyyaml) or ruby. ({e})")


# Paths that are real routes but not part of the public integration surface —
# founder/operator-only endpoints we don't advertise to customers' AI agents.
EXCLUDE_PATHS = {"/platform/metrics"}


def endpoints_by_tag(spec):
    """Return {tag: [(method, path, op), ...]} preserving the spec's tag order."""
    order = [t["name"] for t in spec.get("tags", [])]
    groups = {t: [] for t in order}
    for path, item in spec.get("paths", {}).items():
        if path in EXCLUDE_PATHS:
            continue
        for method in ("get", "post", "put", "patch", "delete"):
            op = item.get(method)
            if not op:
                continue
            tag = (op.get("tags") or ["Other"])[0]
            groups.setdefault(tag, [])
            if tag not in order:
                order.append(tag)
            groups[tag].append((method.upper(), path, op))
    return order, groups


def gen_index(spec):
    """The concise llms.txt."""
    L = []
    L.append("# Recurso")
    L.append("")
    L.append(f"> {TAGLINE}")
    L.append("")
    L.append("This file helps AI coding assistants and agents call the Recurso REST "
             "API correctly. For the complete request/response JSON Schemas, load "
             "https://recurso.dev/llms-full.txt.")
    L.append("")
    L.append("## Conventions")
    L.append("")
    L.append(CONVENTIONS)
    L.append("")
    L.append("## Documentation")
    L.append("")
    for name, url, desc in LINKS:
        L.append(f"- [{name}]({url}): {desc}")
    L.append("")
    L.append("## SDKs")
    L.append("")
    for name, url in SDKS:
        L.append(f"- [{name}]({url})")
    L.append("")
    L.append("## API endpoints")
    L.append("")
    order, groups = endpoints_by_tag(spec)
    for tag in order:
        ops = groups.get(tag)
        if not ops:
            continue
        L.append(f"### {tag}")
        for method, path, op in ops:
            summary = op.get("summary") or op.get("operationId") or ""
            L.append(f"- `{method} {path}`" + (f" — {summary}" if summary else ""))
        L.append("")
    return "\n".join(L).rstrip() + "\n"


def _params(op):
    lines = []
    for p in op.get("parameters", []):
        loc = p.get("in", "")
        req = " (required)" if p.get("required") else ""
        sch = p.get("schema", {})
        typ = sch.get("type", "")
        desc = p.get("description", "")
        lines.append(f"  - `{p.get('name')}` ({loc}{', ' + typ if typ else ''}){req}"
                     + (f": {desc}" if desc else ""))
    return lines


def _body_schema_name(op):
    body = op.get("requestBody", {})
    content = body.get("content", {}).get("application/json", {})
    sch = content.get("schema", {})
    if "$ref" in sch:
        return sch["$ref"].split("/")[-1]
    return None if not sch else "(inline)"


def gen_full(spec):
    """The complete llms-full.txt with JSON Schemas."""
    L = []
    L.append("# Recurso API — full machine reference")
    L.append("")
    L.append(f"> {TAGLINE}")
    L.append("")
    L.append("Generated from the OpenAPI 3.1 spec. Every endpoint is listed with its "
             "parameters and the name of its request/response object; every object is "
             "defined as a JSON Schema in the **Schemas** section at the end. `$ref` "
             "values like `#/components/schemas/Plan` point to those schemas.")
    L.append("")
    L.append("## Conventions")
    L.append("")
    L.append(CONVENTIONS)
    L.append("")
    L.append("## Endpoints")
    L.append("")
    order, groups = endpoints_by_tag(spec)
    for tag in order:
        ops = groups.get(tag)
        if not ops:
            continue
        L.append(f"### {tag}")
        L.append("")
        for method, path, op in ops:
            summary = op.get("summary") or op.get("operationId") or ""
            L.append(f"#### `{method} {path}`" + (f" — {summary}" if summary else ""))
            desc = op.get("description")
            if desc:
                L.append("")
                L.append(desc.strip())
            params = _params(op)
            if params:
                L.append("")
                L.append("Parameters:")
                L.extend(params)
            bn = _body_schema_name(op)
            if bn:
                L.append("")
                L.append(f"Request body (application/json): `{bn}`")
            resp = op.get("responses", {})
            codes = [c for c in resp if c not in ("default",)]
            if codes:
                pieces = []
                for c in sorted(codes):
                    rc = resp[c].get("content", {}).get("application/json", {}).get("schema", {})
                    ref = rc.get("$ref", "").split("/")[-1] if isinstance(rc, dict) else ""
                    pieces.append(f"`{c}`" + (f" → `{ref}`" if ref else ""))
                L.append("")
                L.append("Responses: " + ", ".join(pieces))
            L.append("")
    L.append("## Schemas")
    L.append("")
    L.append("Each object below is a JSON Schema. Field-level `$ref`s reference other "
             "schemas in this section by name.")
    L.append("")
    schemas = spec.get("components", {}).get("schemas", {})
    for name in sorted(schemas):
        L.append(f"### {name}")
        L.append("")
        L.append("```json")
        L.append(json.dumps(schemas[name], indent=2, ensure_ascii=False))
        L.append("```")
        L.append("")
    return "\n".join(L).rstrip() + "\n"


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    default_spec = os.path.join(here, "..", "cmd", "api", "openapi.yaml")
    ap = argparse.ArgumentParser()
    ap.add_argument("--spec", default=default_spec)
    ap.add_argument("--out", default=".")
    args = ap.parse_args()

    spec = load_spec(args.spec)
    os.makedirs(args.out, exist_ok=True)

    idx = gen_index(spec)
    full = gen_full(spec)
    with open(os.path.join(args.out, "llms.txt"), "w") as f:
        f.write(idx)
    with open(os.path.join(args.out, "llms-full.txt"), "w") as f:
        f.write(full)

    n_paths = len(spec.get("paths", {}))
    n_sch = len(spec.get("components", {}).get("schemas", {}))
    print(f"Wrote llms.txt ({len(idx)} bytes) and llms-full.txt ({len(full)} bytes) "
          f"to {args.out} — {n_paths} paths, {n_sch} schemas.")


if __name__ == "__main__":
    main()
