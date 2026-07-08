package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SSOConnection is a tenant's SAML IdP configuration. There is at most one per
// tenant (tenant_id is unique). It is created disabled; SP-initiated login and
// the ACS endpoint only work once Enabled is true and the IdP fields are set.
type SSOConnection struct {
	ID uuid.UUID `db:"id"`
	// TenantID owns the connection; SP endpoints are addressed per-tenant.
	TenantID uuid.UUID `db:"tenant_id"`
	// IDPMetadataXML, when non-empty, is the IdP's full metadata document and
	// takes precedence over the discrete fields below.
	IDPMetadataXML string `db:"idp_metadata_xml"`
	// IDPEntityID is the IdP's SAML entity id (issuer).
	IDPEntityID string `db:"idp_entity_id"`
	// IDPSSOURL is the IdP's HTTP-Redirect SingleSignOnService location.
	IDPSSOURL string `db:"idp_sso_url"`
	// IDPCertificate is the IdP's base64/PEM X.509 signing certificate.
	IDPCertificate string `db:"idp_certificate"`
	// Enabled gates the SP login/ACS endpoints for this tenant.
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Configured reports whether the connection has enough IdP detail to attempt a
// login: either raw metadata XML, or all three discrete fields.
func (c *SSOConnection) Configured() bool {
	if c == nil {
		return false
	}
	if c.IDPMetadataXML != "" {
		return true
	}
	return c.IDPEntityID != "" && c.IDPSSOURL != "" && c.IDPCertificate != ""
}

// SSO domain errors.
var (
	// ErrSSOConnectionNotFound is returned when a tenant has no SSO connection.
	ErrSSOConnectionNotFound = errors.New("sso connection not found")
	// ErrSSONotEnabled is returned when a tenant's connection exists but is
	// disabled or not fully configured. Surfaces as 404 on the public endpoints.
	ErrSSONotEnabled = errors.New("sso is not enabled for this tenant")
	// ErrSSOUserNotFound is returned when a validated assertion's email does not
	// match any existing user in the tenant. This phase does NOT provision users
	// just-in-time, so this surfaces as 403.
	ErrSSOUserNotFound = errors.New("no user in this tenant matches the SSO identity")
	// ErrSSOInvalidAssertion is returned when the SAMLResponse fails validation
	// (signature, audience, timing) or carries no usable email.
	ErrSSOInvalidAssertion = errors.New("invalid SAML assertion")
)
