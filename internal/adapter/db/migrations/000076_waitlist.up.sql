-- Recurso Cloud waitlist (ENG-12): demand capture from the marketing site.
-- Platform-level (no tenant): these are prospects, not tenants' customers.
CREATE TABLE waitlist_signups (
    id         UUID PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    company    TEXT NOT NULL DEFAULT '',
    source     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
