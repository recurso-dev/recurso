-- Drop AI Churn Prediction fields
DROP INDEX IF EXISTS idx_customers_risk_score;
ALTER TABLE customers DROP COLUMN IF EXISTS risk_factors;
ALTER TABLE customers DROP COLUMN IF EXISTS risk_score;
