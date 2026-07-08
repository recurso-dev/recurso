package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// commonEmailAttributes are the SAML attribute names IdPs commonly use to carry
// the user's email address, tried in order before falling back to the NameID.
var commonEmailAttributes = []string{
	"email",
	"emailaddress",
	"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
	"urn:oid:0.9.2342.19200300.100.1.3", // mail
	"http://schemas.xmlsoap.org/claims/emailaddress",
}

// SSOService owns per-tenant SAML SSO: IdP connection config (CRUD) plus the SP
// side (metadata, AuthnRequest, ACS validation + email→user mapping). It holds
// the SP's own key/cert (one per install) and the public base URL used to build
// the per-tenant SP endpoint URLs.
type SSOService struct {
	connections port.SSOConnectionRepository
	users       port.UserRepository
	spKey       *rsa.PrivateKey
	spCert      *x509.Certificate
	baseURL     string // API public base, e.g. https://api.example.com
}

// NewSSOService builds the service. baseURL is the API's public base; SP
// metadata/ACS URLs are derived per-tenant from it. key/cert are the SP signing
// material (see LoadOrGenerateSPKeyPair).
func NewSSOService(connections port.SSOConnectionRepository, users port.UserRepository, key *rsa.PrivateKey, cert *x509.Certificate, baseURL string) *SSOService {
	return &SSOService{
		connections: connections,
		users:       users,
		spKey:       key,
		spCert:      cert,
		baseURL:     strings.TrimRight(baseURL, "/"),
	}
}

// --- connection config (tenant-scoped) ---

// GetConnection returns the tenant's SSO connection or
// domain.ErrSSOConnectionNotFound.
func (s *SSOService) GetConnection(ctx context.Context, tenantID uuid.UUID) (*domain.SSOConnection, error) {
	return s.connections.GetByTenant(ctx, tenantID)
}

// UpsertConnectionInput carries the writable fields of a tenant's connection.
type UpsertConnectionInput struct {
	IDPMetadataXML string
	IDPEntityID    string
	IDPSSOURL      string
	IDPCertificate string
	Enabled        bool
}

// UpsertConnection inserts or updates the tenant's IdP config. Enabling a
// connection that is not fully configured is rejected.
func (s *SSOService) UpsertConnection(ctx context.Context, tenantID uuid.UUID, in UpsertConnectionInput) (*domain.SSOConnection, error) {
	conn := &domain.SSOConnection{
		ID:             uuid.New(),
		TenantID:       tenantID,
		IDPMetadataXML: strings.TrimSpace(in.IDPMetadataXML),
		IDPEntityID:    strings.TrimSpace(in.IDPEntityID),
		IDPSSOURL:      strings.TrimSpace(in.IDPSSOURL),
		IDPCertificate: strings.TrimSpace(in.IDPCertificate),
		Enabled:        in.Enabled,
	}
	// Preserve the existing row's id/created_at when updating.
	if existing, err := s.connections.GetByTenant(ctx, tenantID); err == nil {
		conn.ID = existing.ID
		conn.CreatedAt = existing.CreatedAt
	}
	if conn.Enabled && !conn.Configured() {
		return nil, fmt.Errorf("cannot enable SSO: provide idp metadata XML, or all of entity id, SSO URL and certificate")
	}
	if err := s.connections.Upsert(ctx, conn); err != nil {
		return nil, err
	}
	return s.connections.GetByTenant(ctx, tenantID)
}

// DeleteConnection removes the tenant's connection.
func (s *SSOService) DeleteConnection(ctx context.Context, tenantID uuid.UUID) error {
	return s.connections.Delete(ctx, tenantID)
}

// --- SP endpoints ---

func (s *SSOService) metadataURL(tenantID uuid.UUID) url.URL {
	u, _ := url.Parse(fmt.Sprintf("%s/auth/saml/%s/metadata", s.baseURL, tenantID))
	return *u
}

func (s *SSOService) acsURL(tenantID uuid.UUID) url.URL {
	u, _ := url.Parse(fmt.Sprintf("%s/auth/saml/%s/acs", s.baseURL, tenantID))
	return *u
}

// SPMetadataURL returns the tenant's public SP metadata URL (for display in the
// admin UI so the operator can hand it to their IdP).
func (s *SSOService) SPMetadataURL(tenantID uuid.UUID) string {
	u := s.metadataURL(tenantID)
	return u.String()
}

// SPACSURL returns the tenant's public Assertion Consumer Service URL.
func (s *SSOService) SPACSURL(tenantID uuid.UUID) string {
	u := s.acsURL(tenantID)
	return u.String()
}

// serviceProvider builds a crewjam ServiceProvider for the tenant. When idp is
// true the IdP metadata is attached (required for login/ACS); metadata-only
// callers pass false since SP metadata does not depend on the IdP.
func (s *SSOService) serviceProvider(conn *domain.SSOConnection, withIDP bool) (*saml.ServiceProvider, error) {
	metaURL := s.metadataURL(conn.TenantID)
	sp := &saml.ServiceProvider{
		Key:         s.spKey,
		Certificate: s.spCert,
		MetadataURL: metaURL,
		AcsURL:      s.acsURL(conn.TenantID),
		EntityID:    metaURL.String(),
	}
	if withIDP {
		idp, err := buildIDPMetadata(conn)
		if err != nil {
			return nil, err
		}
		sp.IDPMetadata = idp
	}
	return sp, nil
}

// Metadata returns the tenant's SP metadata XML. The tenant must have a
// connection row (created via the admin API); otherwise
// domain.ErrSSOConnectionNotFound.
func (s *SSOService) Metadata(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	conn, err := s.connections.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sp, err := s.serviceProvider(conn, false)
	if err != nil {
		return nil, err
	}
	return xml.MarshalIndent(sp.Metadata(), "", "  ")
}

// LoginRedirectURL returns the IdP redirect URL for an SP-initiated login. It
// requires the connection to be enabled and configured (else
// domain.ErrSSONotEnabled).
func (s *SSOService) LoginRedirectURL(ctx context.Context, tenantID uuid.UUID) (string, error) {
	conn, err := s.enabledConnection(ctx, tenantID)
	if err != nil {
		return "", err
	}
	sp, err := s.serviceProvider(conn, true)
	if err != nil {
		return "", err
	}
	authURL, err := sp.MakeRedirectAuthenticationRequest("")
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrSSOInvalidAssertion, err)
	}
	return authURL.String(), nil
}

// ProcessACS validates the SAMLResponse on req against the tenant's IdP,
// extracts the email, and maps it to an EXISTING user in the tenant (no JIT
// provisioning). Returns domain.ErrSSOUserNotFound (→403) for an unknown email.
func (s *SSOService) ProcessACS(ctx context.Context, tenantID uuid.UUID, req *http.Request) (*domain.User, error) {
	conn, err := s.enabledConnection(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sp, err := s.serviceProvider(conn, true)
	if err != nil {
		return nil, err
	}
	assertion, err := sp.ParseResponse(req, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrSSOInvalidAssertion, err)
	}
	email := extractAssertionEmail(assertion)
	if email == "" {
		return nil, fmt.Errorf("%w: assertion carries no email", domain.ErrSSOInvalidAssertion)
	}
	return s.MapEmailToUser(ctx, tenantID, email)
}

// MapEmailToUser resolves a validated SSO email to an existing user in the
// tenant. Split out so the mapping/gating is unit-testable without a live IdP
// signature round-trip. Unknown email → domain.ErrSSOUserNotFound.
func (s *SSOService) MapEmailToUser(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrSSOUserNotFound
	}
	// Enforce tenant isolation: the matched user must belong to this tenant.
	if user.TenantID != tenantID {
		return nil, domain.ErrSSOUserNotFound
	}
	return user, nil
}

// enabledConnection returns the tenant's connection only if it exists, is
// enabled AND configured; otherwise domain.ErrSSONotEnabled (→404).
func (s *SSOService) enabledConnection(ctx context.Context, tenantID uuid.UUID) (*domain.SSOConnection, error) {
	conn, err := s.connections.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, domain.ErrSSONotEnabled
	}
	if !conn.Enabled || !conn.Configured() {
		return nil, domain.ErrSSONotEnabled
	}
	return conn, nil
}

// --- IdP metadata construction ---

// buildIDPMetadata produces a crewjam EntityDescriptor for the IdP. If raw
// metadata XML is present it is parsed directly; otherwise the descriptor is
// assembled from the discrete entity-id / SSO-url / certificate fields.
func buildIDPMetadata(conn *domain.SSOConnection) (*saml.EntityDescriptor, error) {
	if conn.IDPMetadataXML != "" {
		md, err := samlsp.ParseMetadata([]byte(conn.IDPMetadataXML))
		if err != nil {
			return nil, fmt.Errorf("%w: bad IdP metadata XML: %v", domain.ErrSSOInvalidAssertion, err)
		}
		return md, nil
	}
	certB64, err := normalizeCertB64(conn.IDPCertificate)
	if err != nil {
		return nil, err
	}
	return &saml.EntityDescriptor{
		EntityID: conn.IDPEntityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
						KeyDescriptors: []saml.KeyDescriptor{
							{
								Use: "signing",
								KeyInfo: saml.KeyInfo{
									X509Data: saml.X509Data{
										X509Certificates: []saml.X509Certificate{
											{Data: certB64},
										},
									},
								},
							},
						},
					},
				},
				SingleSignOnServices: []saml.Endpoint{
					{
						Binding:  saml.HTTPRedirectBinding,
						Location: conn.IDPSSOURL,
					},
				},
			},
		},
	}, nil
}

// normalizeCertB64 accepts a PEM certificate or bare base64 DER and returns the
// base64 DER body (no PEM markers, no whitespace) that crewjam expects.
func normalizeCertB64(cert string) (string, error) {
	cert = strings.TrimSpace(cert)
	if cert == "" {
		return "", fmt.Errorf("%w: empty IdP certificate", domain.ErrSSOInvalidAssertion)
	}
	if strings.Contains(cert, "-----BEGIN") {
		block, _ := pem.Decode([]byte(cert))
		if block == nil {
			return "", fmt.Errorf("%w: undecodable PEM certificate", domain.ErrSSOInvalidAssertion)
		}
		return base64.StdEncoding.EncodeToString(block.Bytes), nil
	}
	// Bare base64: strip whitespace/newlines.
	replacer := strings.NewReplacer("\n", "", "\r", "", " ", "", "\t", "")
	return replacer.Replace(cert), nil
}

// extractAssertionEmail pulls an email from the assertion's attribute
// statements (common names) or falls back to the NameID when it looks like an
// email address.
func extractAssertionEmail(assertion *saml.Assertion) string {
	if assertion == nil {
		return ""
	}
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			key := strings.ToLower(strings.TrimSpace(attr.Name))
			friendly := strings.ToLower(strings.TrimSpace(attr.FriendlyName))
			for _, want := range commonEmailAttributes {
				if key == want || friendly == want {
					for _, v := range attr.Values {
						if strings.Contains(v.Value, "@") {
							return strings.TrimSpace(v.Value)
						}
					}
				}
			}
		}
	}
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		if v := strings.TrimSpace(assertion.Subject.NameID.Value); strings.Contains(v, "@") {
			return v
		}
	}
	return ""
}

// --- SP key/cert material ---

// LoadOrGenerateSPKeyPair returns the SP signing key + certificate. If both
// keyPEM and certPEM are provided (SAML_SP_KEY / SAML_SP_CERT), they are parsed;
// otherwise an ephemeral self-signed pair is generated at boot (fine for
// bringing the SP up and rendering metadata; a stable env-provided pair is
// recommended for a real IdP integration so the SP cert does not change on
// restart).
func LoadOrGenerateSPKeyPair(keyPEM, certPEM string) (*rsa.PrivateKey, *x509.Certificate, error) {
	keyPEM = strings.TrimSpace(keyPEM)
	certPEM = strings.TrimSpace(certPEM)
	if keyPEM != "" && certPEM != "" {
		key, err := parseRSAPrivateKey(keyPEM)
		if err != nil {
			return nil, nil, err
		}
		cert, err := parseCertificate(certPEM)
		if err != nil {
			return nil, nil, err
		}
		return key, cert, nil
	}
	return generateSelfSignedSPKeyPair()
}

func parseRSAPrivateKey(keyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, fmt.Errorf("SAML_SP_KEY is not valid PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("SAML_SP_KEY: unsupported private key: %w", err)
	}
	rsaKey, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("SAML_SP_KEY must be an RSA private key")
	}
	return rsaKey, nil
}

func parseCertificate(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("SAML_SP_CERT is not valid PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("SAML_SP_CERT: %w", err)
	}
	return cert, nil
}

func generateSelfSignedSPKeyPair() (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "recurso-saml-sp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, nil
}
