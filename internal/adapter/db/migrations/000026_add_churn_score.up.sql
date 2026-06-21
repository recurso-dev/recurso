-- Add AI Churn Prediction fields to customers table
ALTER TABLE customers ADD COLUMN IF NOT EXISTS risk_score INT DEFAULT 0;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS risk_factors JSONB;

-- Create index for high-risk queries
CREATE INDEX IF NOT EXISTS idx_customers_risk_score ON customers(risk_score);
