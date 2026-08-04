package domain

// ctxKey is a private type for context keys defined in this package to avoid
// collisions with keys defined in other packages (staticcheck SA1029).
type ctxKey string

// TenantIDKey is the context key used to pass the tenant ID between the
// HTTP handlers and the repository layer.
const TenantIDKey ctxKey = "tenant_id"

// RequestIDKey and UserIDKey carry the request and authenticated-user IDs on
// the context so a context-aware slog handler can stamp every downstream log
// line with them — making a production incident reconstructable across the
// middleware → handler → service chain.
const (
	RequestIDKey ctxKey = "request_id"
	UserIDKey    ctxKey = "user_id"
)
