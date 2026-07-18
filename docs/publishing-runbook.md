# Publishing Runbook (founder-only 🔒)

These three steps make the SDKs and image usable by the world. They need
**your** credentials (npm, PyPI, GitHub), so run them yourself — Claude can
prepare and verify the packages but cannot authenticate or publish.

Everything below is already built, tested, and CI-gated. Nothing here changes
code; it only publishes what exists.

---

## 1. npm — `recurso` ✅ DONE (2026-07-18)

Published as [`recurso`](https://www.npmjs.com/package/recurso) v1.2.0 (the
npm name is `recurso`; the repo stays
[recurso-node](https://github.com/recurso-dev/recurso-node), local checkout
`../recurso-node`). Owned by the `team-recurso` org.

To publish a new version:

```bash
cd ../recurso-node
# bump "version" in package.json, then:
npm publish --otp=<code>        # prepublishOnly runs the build
npm view recurso version        # verify
```

---

## 2. PyPI — `recurso`

The Python SDK now lives in its own repo:
[recurso-python](https://github.com/recurso-dev/recurso-python)
(local checkout: `../recurso-python`).

```bash
cd ../recurso-python

# a) Is the name free? Open https://pypi.org/project/recurso/ — 404 = available.
#    (If taken, change `name = "recurso"` in pyproject.toml to e.g. "recurso-sdk".)

# b) Tools (one-time)
pip install build twine

# c) Build the distribution
python3 -m build          # produces dist/*.whl and dist/*.tar.gz

# d) Upload (prompts for a PyPI API token — create one at pypi.org → Account → API tokens)
python3 -m twine upload dist/*

# e) Verify (in a fresh venv)
pip install recurso
```

---

## 3. GHCR — make the image public

The image already publishes to `ghcr.io/recurso-dev/recurso` on every push to
`main`; it's just **private** by default.

1. GitHub → your profile/org → **Packages** → `recur-so`
2. **Package settings** → **Danger Zone** → **Change visibility** → **Public**
3. Verify from a logged-out shell:
   ```bash
   docker logout ghcr.io
   docker pull ghcr.io/recurso-dev/recurso:latest   # should succeed without auth
   ```

---

## After publishing — tell Claude

Once these are live, ask Claude to:
- Flip the docs' "Not yet published" notes to `npm install recurso` /
  `pip install recurso`, and add real install badges to the READMEs.
  (npm side done 2026-07-18.)
- Update the `examples/nextjs-starter` to use the published SDK instead of plain
  `fetch` (optional).
- Check the three Track 3 / Lane 3 roadmap boxes (npm publish, GHCR public).

## Notes / gotchas

- **2FA/OTP:** npm and PyPI may prompt for a one-time code at publish time.
- **Versions are immutable:** you can't re-publish `1.1.0` / `1.0.0` — bump the
  version to republish.
- **The Docker image needs no code change** to go public — only the visibility
  toggle. It's the fastest of the three (~30 seconds).
