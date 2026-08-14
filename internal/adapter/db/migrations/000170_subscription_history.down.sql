DROP TRIGGER IF EXISTS subscription_history_trg ON subscriptions;
DROP FUNCTION IF EXISTS record_subscription_change();
DROP TABLE IF EXISTS subscription_history;
