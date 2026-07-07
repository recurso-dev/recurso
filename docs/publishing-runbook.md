# Publishing Runbook (founder-only 🔒)

These three steps make the SDKs and image usable by the world. They need
**your** credentials (npm, PyPI, GitHub), so run them yourself — Claude can
prepare and verify the packages but cannot authenticate or publish.

Everything below is already built, tested, and CI-gated. Nothing here changes
code; it only publishes what exists.

---

## 1. npm — `recurso-node`

```bash
cd sdk/node

# a) Is the name free? (404 = available; if it prints a package, see "Name taken" below)
npm view recurso-node

# b) Log in (opens browser / prompts for OTP if 2FA is on)
npm login

# c) Build fresh and publish PUBLICLY (unscoped packages need --access public)
npm run build
npm publish --access public

# d) Verify
npm view recurso-node version   # should print 1.1.0
```

**Name taken?** Rename to a scope you own: set `"name": "@your-org/recurso"` in
`sdk/node/package.json`, then `npm publish --access public`.

---

## 2. PyPI — `recurso`

```bash
cd sdk/python

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

The image already publishes to `ghcr.io/swapnull-in/recur-so` on every push to
`main`; it's just **private** by default.

1. GitHub → your profile/org → **Packages** → `recur-so`
2. **Package settings** → **Danger Zone** → **Change visibility** → **Public**
3. Verify from a logged-out shell:
   ```bash
   docker logout ghcr.io
   docker pull ghcr.io/swapnull-in/recur-so:latest   # should succeed without auth
   ```

---

## After publishing — tell Claude

Once these are live, ask Claude to:
- Flip the docs' "Not yet published to npm" notes to `npm install recurso-node` /
  `pip install recurso`, and add real install badges to the READMEs.
- Update the `examples/nextjs-starter` to use the published SDK instead of plain
  `fetch` (optional).
- Check the three Track 3 / Lane 3 roadmap boxes (npm publish, GHCR public).

## Notes / gotchas

- **2FA/OTP:** npm and PyPI may prompt for a one-time code at publish time.
- **Versions are immutable:** you can't re-publish `1.1.0` / `1.0.0` — bump the
  version to republish.
- **The Docker image needs no code change** to go public — only the visibility
  toggle. It's the fastest of the three (~30 seconds).
