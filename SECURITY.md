# Security Policy

## Reporting a vulnerability

If you believe you have found a security vulnerability in Recurso, please report
it **privately**. Do not open a public GitHub issue, pull request, or discussion
for security matters.

Email **swapnil.go20@gmail.com** with:

- a description of the issue and the impact you believe it has,
- the steps or a proof-of-concept needed to reproduce it,
- the affected version, commit, or endpoint, and
- any suggested remediation, if you have one.

You will receive an acknowledgement within **3 business days**. We will keep you
updated on the fix and coordinate a disclosure timeline with you. We ask that you
give us a reasonable window to release a fix before any public disclosure. We do
not currently run a paid bug-bounty program, but we credit reporters in the
release notes unless you prefer to remain anonymous.

## Supported versions

Recurso is pre-1.0 and moves quickly. Security fixes land on `main` and in the
**latest released minor version**. Older tags do not receive backported fixes;
please track the latest release.

## Handling secrets

Recurso never commits real credentials. Configuration (database URLs, gateway
keys, SAML material) is supplied at runtime via environment variables — see
`.env.example` for the full list. If you are self-hosting, keep your `.env` out
of version control (it is gitignored) and rotate any key you suspect has been
exposed.
