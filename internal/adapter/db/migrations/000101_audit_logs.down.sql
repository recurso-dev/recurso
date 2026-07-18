DROP TRIGGER IF EXISTS trg_audit_logs_immutable ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_immutable();
DROP TABLE IF EXISTS audit_logs;
