-- US economic-nexus thresholds (Phase 2: threshold tracking).
--
-- COMPLIANCE BOUNDARY: this dataset encodes each state's commonly-cited
-- post-Wayfair economic-nexus thresholds (most: $100,000 sales OR 200
-- transactions; several states differ). It is seeded UNCERTIFIED
-- (certified = FALSE) and must be reviewed by a US sales-tax professional or
-- replaced by a maintained provider dataset before being relied on for
-- filing — the same liability class as the GST/HSN rate map (see
-- docs/design-us-nexus.md). The engine surfaces dataset_certified so callers
-- can display the caveat.
--
-- measurement_period is simplified to the calendar year for all states in
-- this seed; the certification pass refines states that use rolling-12-month
-- or previous-year measurements.
CREATE TABLE us_nexus_thresholds (
    state_code         CHAR(2) PRIMARY KEY,
    sales_threshold    BIGINT,      -- USD cents; NULL = no sales threshold
    txn_threshold      INTEGER,     -- NULL = no transaction-count threshold
    combinator         TEXT NOT NULL DEFAULT 'or'
                       CHECK (combinator IN ('or', 'and')),
    measurement_period TEXT NOT NULL DEFAULT 'calendar_year',
    certified          BOOLEAN NOT NULL DEFAULT FALSE,
    notes              TEXT,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- States with no state-level sales tax (DE, MT, NH, OR) have no row: no
-- threshold to track. AK has no state tax but ARSSTC local economic nexus.
INSERT INTO us_nexus_thresholds (state_code, sales_threshold, txn_threshold, combinator, notes) VALUES
    ('AK', 10000000, NULL, 'or', 'ARSSTC local nexus; txn count repealed 2025'),
    ('AL', 25000000, NULL, 'or', ''),
    ('AR', 10000000, 200,  'or', ''),
    ('AZ', 10000000, NULL, 'or', ''),
    ('CA', 50000000, NULL, 'or', ''),
    ('CO', 10000000, NULL, 'or', ''),
    ('CT', 10000000, 200,  'and', ''),
    ('DC', 10000000, 200,  'or', ''),
    ('FL', 10000000, NULL, 'or', ''),
    ('GA', 10000000, 200,  'or', ''),
    ('HI', 10000000, 200,  'or', ''),
    ('IA', 10000000, NULL, 'or', ''),
    ('ID', 10000000, NULL, 'or', ''),
    ('IL', 10000000, 200,  'or', ''),
    ('IN', 10000000, NULL, 'or', 'txn count repealed 2024'),
    ('KS', 10000000, NULL, 'or', ''),
    ('KY', 10000000, 200,  'or', ''),
    ('LA', 10000000, NULL, 'or', 'txn count repealed 2023'),
    ('MA', 10000000, NULL, 'or', ''),
    ('MD', 10000000, 200,  'or', ''),
    ('ME', 10000000, NULL, 'or', ''),
    ('MI', 10000000, 200,  'or', ''),
    ('MN', 10000000, 200,  'or', ''),
    ('MO', 10000000, NULL, 'or', ''),
    ('MS', 25000000, NULL, 'or', ''),
    ('NC', 10000000, NULL, 'or', ''),
    ('ND', 10000000, NULL, 'or', ''),
    ('NE', 10000000, 200,  'or', ''),
    ('NJ', 10000000, 200,  'or', ''),
    ('NM', 10000000, NULL, 'or', ''),
    ('NV', 10000000, 200,  'or', ''),
    ('NY', 50000000, 100,  'and', ''),
    ('OH', 10000000, 200,  'or', ''),
    ('OK', 10000000, NULL, 'or', ''),
    ('PA', 10000000, NULL, 'or', ''),
    ('RI', 10000000, 200,  'or', ''),
    ('SC', 10000000, NULL, 'or', ''),
    ('SD', 10000000, NULL, 'or', 'txn count repealed 2023'),
    ('TN', 10000000, NULL, 'or', ''),
    ('TX', 50000000, NULL, 'or', ''),
    ('UT', 10000000, 200,  'or', ''),
    ('VA', 10000000, 200,  'or', ''),
    ('VT', 10000000, 200,  'or', ''),
    ('WA', 10000000, NULL, 'or', ''),
    ('WI', 10000000, NULL, 'or', ''),
    ('WV', 10000000, 200,  'or', ''),
    ('WY', 10000000, NULL, 'or', 'txn count repealed 2024');
