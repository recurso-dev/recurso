package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// NotificationService handles sending notifications for billing events
type NotificationService struct {
	emailSender port.EmailSender
	// baseURL is the API base (e.g. https://api.recurso.dev) — the hosted
	// checkout at /checkout/{id} is served here.
	baseURL string
	// portalBaseURL is where the customer-facing portal SPA is served (PORTAL_URL,
	// e.g. https://app.recurso.dev). Portal links (login / update payment method)
	// MUST use this, not the API base. Defaults to baseURL until SetPortalBaseURL.
	portalBaseURL string
}

func NewNotificationService(emailSender port.EmailSender, baseURL string) *NotificationService {
	return &NotificationService{
		emailSender:   emailSender,
		baseURL:       baseURL,
		portalBaseURL: baseURL,
	}
}

// SetPortalBaseURL points customer-facing portal links at the portal SPA
// (PORTAL_URL) instead of the API domain.
func (s *NotificationService) SetPortalBaseURL(url string) {
	if url != "" {
		s.portalBaseURL = url
	}
}

// InvoiceData for invoice emails
type InvoiceData struct {
	CustomerName  string
	CustomerEmail string
	InvoiceNumber string
	InvoiceID     string // used to build the hosted-checkout link when PaymentURL is empty
	Amount        string
	DueDate       string
	PaymentURL    string
}

// SendInvoiceCreated sends an invoice notification
func (s *NotificationService) SendInvoiceCreated(ctx context.Context, data InvoiceData) error {
	// The template renders a "Pay Now" button — an empty href is a dead
	// button, so default it to the hosted checkout for this invoice.
	if data.PaymentURL == "" && data.InvoiceID != "" {
		data.PaymentURL = strings.TrimRight(s.baseURL, "/") + "/checkout/" + data.InvoiceID
	}

	content, err := s.renderTemplate(email.InvoiceCreatedTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("New Invoice", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  fmt.Sprintf("Invoice %s - %s Due", data.InvoiceNumber, data.Amount),
		HTMLBody: html,
	})
}

// PaymentData for payment emails
type PaymentData struct {
	CustomerName  string
	CustomerEmail string
	InvoiceNumber string
	Amount        string
	PaymentDate   string
}

// SendPaymentReceived sends a payment confirmation
func (s *NotificationService) SendPaymentReceived(ctx context.Context, data PaymentData) error {
	content, err := s.renderTemplate(email.PaymentReceivedTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Payment Received", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  fmt.Sprintf("Payment Received - %s", data.Amount),
		HTMLBody: html,
	})
}

// SubscriptionData for subscription emails
type SubscriptionData struct {
	CustomerName    string
	CustomerEmail   string
	PlanName        string
	Price           string
	Interval        string
	StartDate       string
	NextBillingDate string
	PortalURL       string
}

// SendSubscriptionCreated sends a subscription confirmation
func (s *NotificationService) SendSubscriptionCreated(ctx context.Context, data SubscriptionData) error {
	content, err := s.renderTemplate(email.SubscriptionCreatedTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Subscription Created", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  fmt.Sprintf("Welcome to %s!", data.PlanName),
		HTMLBody: html,
	})
}

// MagicLinkData for login emails
type MagicLinkData struct {
	Email    string
	LoginURL string
}

// SendMagicLink sends a passwordless login link
func (s *NotificationService) SendMagicLink(ctx context.Context, toEmail string, token string) error {
	data := MagicLinkData{
		Email:    toEmail,
		LoginURL: fmt.Sprintf("%s/portal/verify?token=%s", s.baseURL, token),
	}

	content, err := s.renderTemplate(email.MagicLinkTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Login to Recurso", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       toEmail,
		Subject:  "Login to Your Billing Portal",
		HTMLBody: html,
	})
}

// PasswordResetData for admin-dashboard password reset emails.
type PasswordResetData struct {
	ResetURL string
}

// TeamInviteData backs the team-invite email.
type TeamInviteData struct {
	Name      string
	InviteURL string
}

// SendInvite emails a team-invite link to a newly-added teammate. The link
// embeds a single-use token; clicking it lets them set their own password.
func (s *NotificationService) SendInvite(ctx context.Context, toEmail, name, inviteURL string) error {
	content, err := s.renderTemplate(email.TeamInviteTemplate, TeamInviteData{Name: name, InviteURL: inviteURL})
	if err != nil {
		return err
	}
	html, err := s.wrapInBaseTemplate("You've been invited to Recurso", content)
	if err != nil {
		return err
	}
	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       toEmail,
		Subject:  "You've been invited to Recurso",
		HTMLBody: html,
	})
}

// SendPasswordReset emails a password-reset link to a dashboard user. The link
// already embeds the single-use token.
func (s *NotificationService) SendPasswordReset(ctx context.Context, toEmail, resetURL string) error {
	content, err := s.renderTemplate(email.PasswordResetTemplate, PasswordResetData{ResetURL: resetURL})
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Reset your password", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       toEmail,
		Subject:  "Reset your Recurso password",
		HTMLBody: html,
	})
}

// PaymentFailedData for failed payment emails
type PaymentFailedData struct {
	CustomerName     string
	CustomerEmail    string
	InvoiceNumber    string
	Amount           string
	FailureReason    string
	UpdatePaymentURL string
}

// SendPaymentFailed notifies customer of failed payment
func (s *NotificationService) SendPaymentFailed(ctx context.Context, data PaymentFailedData) error {
	// Deep-link the "Update Payment Method" button to the customer portal so a
	// failed payment can be self-served (ENG-5 Phase 3). Without this the button
	// rendered with an empty href.
	if data.UpdatePaymentURL == "" {
		data.UpdatePaymentURL = s.portalPaymentMethodURL()
	}

	content, err := s.renderTemplate(email.PaymentFailedTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Payment Failed", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  "Action Required: Payment Failed",
		HTMLBody: html,
	})
}

// portalPaymentMethodURL is the customer-portal entry a payment-recovery email
// links to. The portal's magic-link login gates it, after which the customer
// updates their card (Stripe) or re-authorizes their mandate. It must be the
// SPA's login route: bare "/portal" matches nothing and the router's catch-all
// bounces the customer to the merchant dashboard.
func (s *NotificationService) portalPaymentMethodURL() string {
	return strings.TrimRight(s.portalBaseURL, "/") + "/portal/login"
}

// Helper to render a template
func (s *NotificationService) renderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Helper to wrap content in base template
func (s *NotificationService) wrapInBaseTemplate(subject string, content string) (string, error) {
	data := struct {
		Subject string
		Content template.HTML
	}{
		Subject: subject,
		Content: template.HTML(content),
	}

	tmpl, err := template.New("base").Parse(email.EmailBaseTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GiftPurchasedData for gift notification emails to recipients
type GiftPurchasedData struct {
	RecipientEmail string
	PlanName       string
	Duration       string
	GiftCode       string
	RedeemURL      string
}

// SendGiftPurchased notifies the recipient that they've received a gift subscription
func (s *NotificationService) SendGiftPurchased(ctx context.Context, data GiftPurchasedData) error {
	content, err := s.renderTemplate(email.GiftPurchasedTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("You've Received a Gift!", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.RecipientEmail,
		Subject:  fmt.Sprintf("You've been gifted a %s subscription!", data.PlanName),
		HTMLBody: html,
	})
}

// SendPreChargeReminder sends 24-hour pre-charge notification (RBI compliance)
func (s *NotificationService) SendPreChargeReminder(ctx context.Context, data email.PreChargeEmailData) error {
	content, err := s.renderTemplate(email.PreChargeReminderTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Upcoming Payment Reminder", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  fmt.Sprintf("Payment Reminder: %s subscription renews tomorrow", data.PlanName),
		HTMLBody: html,
	})
}

// SendTrialEndingReminder notifies a customer that their trial is about to end.
func (s *NotificationService) SendTrialEndingReminder(ctx context.Context, data email.TrialEndingEmailData) error {
	content, err := s.renderTemplate(email.TrialEndingTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Your trial is ending soon", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  fmt.Sprintf("Your %s trial ends on %s", data.PlanName, data.TrialEndDate),
		HTMLBody: html,
	})
}

// SendNexusThresholdAlert emails the tenant that they're approaching or have
// crossed a US state's economic-nexus threshold (Track D · D1). `level` is
// "approaching" or "crossed".
func (s *NotificationService) SendNexusThresholdAlert(ctx context.Context, level string, data email.NexusAlertData) error {
	tmpl := email.NexusApproachingTemplate
	subject := fmt.Sprintf("Heads up: nearing the sales-tax threshold in %s", data.State)
	if level == string(domain.NexusAlertCrossed) {
		tmpl = email.NexusCrossedTemplate
		subject = fmt.Sprintf("Action needed: sales-tax threshold crossed in %s", data.State)
	}

	content, err := s.renderTemplate(tmpl, data)
	if err != nil {
		return err
	}
	html, err := s.wrapInBaseTemplate(subject, content)
	if err != nil {
		return err
	}
	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.RecipientEmail,
		ToName:   data.RecipientName,
		Subject:  subject,
		HTMLBody: html,
	})
}

// SendDunningEmail sends dunning notifications based on escalation level
func (s *NotificationService) SendDunningEmail(ctx context.Context, level int, data email.DunningEmailData) error {
	var tmplStr string
	var subject string

	switch level {
	case 1:
		tmplStr = email.DunningFirstReminderTemplate
		subject = "Action Required: Payment Failed"
	case 2:
		tmplStr = email.DunningSecondReminderTemplate
		subject = "⚠️ Payment Still Pending - Action Required"
	case 3:
		tmplStr = email.DunningFinalNoticeTemplate
		subject = "🚨 Final Notice: Service Suspension"
	default:
		return fmt.Errorf("invalid dunning level: %d", level)
	}

	if data.UpdatePaymentURL == "" {
		data.UpdatePaymentURL = s.portalPaymentMethodURL()
	}

	content, err := s.renderTemplate(tmplStr, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate(subject, content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  subject,
		HTMLBody: html,
	})
}

// SendCardExpiringNotification sends a card expiry warning email
func (s *NotificationService) SendCardExpiringNotification(ctx context.Context, data email.CardExpiringEmailData) error {
	if data.UpdatePaymentURL == "" {
		data.UpdatePaymentURL = s.portalPaymentMethodURL()
	}

	content, err := s.renderTemplate(email.CardExpiringTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Card Expiring Soon", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       data.CustomerEmail,
		ToName:   data.CustomerName,
		Subject:  fmt.Sprintf("Your %s card ending in %s is expiring soon", data.CardBrand, data.CardLast4),
		HTMLBody: html,
	})
}

// SendSubscriptionCancelled sends cancellation confirmation
func (s *NotificationService) SendSubscriptionCancelled(ctx context.Context, customerEmail, customerName, planName, accessUntil, reactivateURL string) error {
	data := struct {
		CustomerName  string
		PlanName      string
		AccessUntil   string
		ReactivateURL string
	}{
		CustomerName:  customerName,
		PlanName:      planName,
		AccessUntil:   accessUntil,
		ReactivateURL: reactivateURL,
	}

	content, err := s.renderTemplate(email.SubscriptionCancelledTemplate, data)
	if err != nil {
		return err
	}

	html, err := s.wrapInBaseTemplate("Subscription Cancelled", content)
	if err != nil {
		return err
	}

	return s.emailSender.Send(ctx, port.EmailMessage{
		To:       customerEmail,
		ToName:   customerName,
		Subject:  "Your subscription has been cancelled",
		HTMLBody: html,
	})
}

// EmailLog tracks sent emails
type EmailLog struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	ToEmail   string
	Subject   string
	EventType string
	SentAt    time.Time
	Status    string
}
