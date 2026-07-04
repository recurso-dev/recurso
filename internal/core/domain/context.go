package domain

// ctxKey is a private type for context keys defined in this package to avoid
// collisions with keys defined in other packages (staticcheck SA1029).
type ctxKey string

// TenantIDKey is the context key used to pass the tenant ID between the
// HTTP handlers and the repository layer.
const TenantIDKey ctxKey = "tenant_id"
