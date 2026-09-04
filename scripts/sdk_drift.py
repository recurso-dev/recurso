#!/usr/bin/env python3
"""Cross-repo drift check: OpenAPI spec vs the three SDKs and the docs copy.

Compares every path in cmd/api/openapi.yaml against the request paths found
in the recurso-go, recurso-node and recurso-python sources, and byte-compares
the spec against the copy the docs site serves.

Usage:
    scripts/sdk_drift.py [--go DIR] [--node DIR] [--python DIR] [--docs DIR]
                         [--baseline FILE] [--update-baseline] [--report]

Exit status is non-zero when:
  * an SDK's covered-path count drops below the recorded baseline (ratchet), or
  * the docs copy of the spec differs from the source spec.

Paths under "browser-flow" prefixes (auth/*, portal/*, checkout/*, webhooks)
are session or redirect flows, not API-key calls, and are excluded from SDK
coverage. They are still counted for the docs copy check.

No third-party dependencies: the spec is scanned line-by-line for top-level
path keys, so this runs on a bare python3.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
SPEC = os.path.join(REPO, "cmd", "api", "openapi.yaml")
DEFAULT_BASELINE = os.path.join(REPO, "scripts", "sdk_drift_baseline.json")

# Prefixes that are browser/session flows rather than SDK surface.
BROWSER_PREFIXES = ("/auth/", "/portal/", "/checkout", "/webhooks/", "/health", "/openapi")

PATH_KEY = re.compile(r"^  (/[^\s:]+):\s*$")
PARAM = re.compile(r"\{[^}]+\}")


def spec_paths(spec_file: str) -> list[str]:
    paths: list[str] = []
    with open(spec_file, encoding="utf-8") as fh:
        in_paths = False
        for line in fh:
            if line.startswith("paths:"):
                in_paths = True
                continue
            if in_paths and line.strip() and not line.startswith(" ") and not line.startswith("#"):
                in_paths = False
            if in_paths:
                m = PATH_KEY.match(line)
                if m:
                    paths.append(m.group(1))
    return sorted(set(paths))


def normalize(path: str) -> str:
    """Collapse every path parameter to {} so styles compare equal."""
    return PARAM.sub("{}", path.rstrip("/"))


def sdk_scope(path: str) -> bool:
    return not path.startswith(BROWSER_PREFIXES)


def read_sources(root: str, exts: tuple[str, ...], skip_dirs=("node_modules", ".git", "dist", "test", "tests")) -> str:
    chunks: list[str] = []
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in skip_dirs]
        for name in filenames:
            if name.endswith(exts) and not name.endswith("_test.go") and not name.endswith(".test.ts") and "schema.d.ts" not in name:
                with open(os.path.join(dirpath, name), encoding="utf-8", errors="replace") as fh:
                    chunks.append(fh.read())
    return "\n".join(chunks)


def go_paths(root: str) -> set[str]:
    src = read_sources(root, (".go",))
    found: set[str] = set()
    # The Go client's base URL already carries /v1; resource paths omit it.
    for m in re.finditer(r'"(/[a-zA-Z0-9_\-/{}%.:]*)"', src):
        p = m.group(1)
        if p in ("/",) or p.startswith("/v1"):
            continue
        p = p.replace("%s", "{}").replace("%d", "{}")
        found.add(normalize("/v1" + p))
        # doRoot() sends the literal without the /v1 prefix (e.g. /version).
        found.add(normalize(p))
    return found


def node_paths(root: str) -> set[str]:
    src = read_sources(os.path.join(root, "src"), (".ts",))
    found: set[str] = set()
    # Any leading-slash string literal: versioned (/v1/...) and the handful of
    # unversioned operational paths (/version, /metrics, /waitlist, ...).
    for m in re.finditer(r"[`'\"](/[a-zA-Z0-9_\-/{}$.:]*)[`'\"]", src):
        p = re.sub(r"\$\{[^}]+\}", "{}", m.group(1))
        found.add(normalize(p))
    return found


def python_paths(root: str) -> set[str]:
    src = read_sources(os.path.join(root, "recurso"), (".py",))
    found: set[str] = set()
    for m in re.finditer(r'"url":\s*"(/[^"]+)"', src):
        found.add(normalize(m.group(1)))
    return found


def compare(spec: list[str], found: set[str]) -> tuple[list[str], list[str]]:
    covered, missing = [], []
    for p in spec:
        if not sdk_scope(p):
            continue
        (covered if normalize(p) in found else missing).append(p)
    return covered, missing


def docs_drift(spec_file: str, docs_root: str) -> list[str]:
    copy = os.path.join(docs_root, "api-reference", "openapi.yaml")
    if not os.path.exists(copy):
        return [f"docs copy not found at {copy}"]
    with open(spec_file, "rb") as a, open(copy, "rb") as b:
        if a.read() == b.read():
            return []
    src, dst = set(spec_paths(spec_file)), set(spec_paths(copy))
    problems = [f"docs api-reference/openapi.yaml differs from cmd/api/openapi.yaml"]
    for p in sorted(src - dst):
        problems.append(f"  missing in docs copy: {p}")
    for p in sorted(dst - src):
        problems.append(f"  only in docs copy:    {p}")
    if len(problems) == 1:
        problems.append("  (same path set; content differs — re-copy the file)")
    return problems


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parent = os.path.dirname(REPO)
    ap.add_argument("--spec", default=SPEC)
    ap.add_argument("--go", default=os.path.join(parent, "recurso-go"))
    ap.add_argument("--node", default=os.path.join(parent, "recurso-node"))
    ap.add_argument("--python", default=os.path.join(parent, "recurso-python"))
    ap.add_argument("--docs", default=os.path.join(parent, "docs"))
    ap.add_argument("--baseline", default=DEFAULT_BASELINE)
    ap.add_argument("--update-baseline", action="store_true", help="write current coverage as the new floor")
    ap.add_argument("--report", action="store_true", help="print every missing path per SDK")
    args = ap.parse_args()

    spec = spec_paths(args.spec)
    in_scope = [p for p in spec if sdk_scope(p)]
    print(f"spec: {len(spec)} paths ({len(in_scope)} in SDK scope)")

    sdks = {
        "go": (args.go, go_paths),
        "node": (args.node, node_paths),
        "python": (args.python, python_paths),
    }
    baseline = {}
    if os.path.exists(args.baseline):
        with open(args.baseline, encoding="utf-8") as fh:
            baseline = json.load(fh)

    failures: list[str] = []
    current: dict[str, int] = {}
    for name, (root, fn) in sdks.items():
        if not os.path.isdir(root):
            print(f"{name}: skipped (no checkout at {root})")
            continue
        covered, missing = compare(spec, fn(root))
        current[name] = len(covered)
        floor = baseline.get(name)
        status = "ok"
        if floor is not None and len(covered) < floor:
            status = f"REGRESSION (baseline {floor})"
            failures.append(f"{name} SDK covers {len(covered)} paths, below baseline {floor}")
        print(f"{name}: {len(covered)}/{len(in_scope)} in-scope paths covered, {len(missing)} missing — {status}")
        if args.report:
            for p in missing:
                print(f"    missing: {p}")

    if os.path.isdir(args.docs):
        problems = docs_drift(args.spec, args.docs)
        if problems:
            failures.append(problems[0])
            print("docs: DRIFT")
            for line in problems:
                print("    " + line)
        else:
            print("docs: in sync")
    else:
        print(f"docs: skipped (no checkout at {args.docs})")

    if args.update_baseline:
        with open(args.baseline, "w", encoding="utf-8") as fh:
            json.dump(current, fh, indent=2, sort_keys=True)
            fh.write("\n")
        print(f"baseline written to {args.baseline}: {current}")

    if failures:
        print("\nFAIL:")
        for f in failures:
            print("  - " + f)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
