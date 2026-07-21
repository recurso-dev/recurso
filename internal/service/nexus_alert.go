package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// nexusApproachingPct is the proximity (percent of a state's economic-nexus
// threshold) at which an "approaching" heads-up is sent, before the crossing.
const nexusApproachingPct = 80

// NexusAlertService turns per-state economic-nexus status into proactive alerts
// to the tenant (Track D · D1). The nexus scheduler already establishes economic
// nexus on a crossing, but silently; this notifies the tenant's owner/admin when
// they near (default 80%) or cross a state's threshold, so a registration
// obligation is never missed. Dedup is per (tenant, state, calendar year, level),
// so a state that stays crossed isn't re-alerted daily.
type NexusAlertService struct {
	status      nexusStatusReader
	store       nexusAlertStore
	users       nexusRecipientLister
	notifier    nexusAlertNotifier
	settingsURL string
	logger      *slog.Logger
}

// Narrow dependencies, for testability.
type nexusStatusReader interface {
	Status(ctx context.Context, tenantID uuid.UUID, year int) ([]domain.NexusStateStatus, error)
}

type nexusAlertStore interface {
	ClaimNexusAlert(ctx context.Context, tenantID uuid.UUID, stateCode string, year int, level string, proximityPct int) (bool, error)
}

type nexusRecipientLister interface {
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error)
}

type nexusAlertNotifier interface {
	SendNexusThresholdAlert(ctx context.Context, level string, data email.NexusAlertData) error
}

func NewNexusAlertService(status nexusStatusReader, store nexusAlertStore, users nexusRecipientLister, notifier nexusAlertNotifier, baseURL string) *NexusAlertService {
	return &NexusAlertService{
		status:      status,
		store:       store,
		users:       users,
		notifier:    notifier,
		settingsURL: strings.TrimRight(baseURL, "/") + "/settings/tax-nexus",
		logger:      slog.Default().With("service", "nexus_alert"),
	}
}

// EvaluateAndAlert computes the tenant's per-state nexus status for the year
// (which also establishes any new economic nexus) and sends an alert for each
// state that has newly reached the approaching band or crossed this year. The
// atomic claim dedups, so this is safe to run daily.
func (s *NexusAlertService) EvaluateAndAlert(ctx context.Context, tenantID uuid.UUID, year int) error {
	statuses, err := s.status.Status(ctx, tenantID, year)
	if err != nil {
		return fmt.Errorf("nexus alert: status: %w", err)
	}

	var recipientEmail, recipientName string
	recipientLoaded := false

	for _, st := range statuses {
		level := levelFor(st, year)
		if level == "" {
			continue
		}

		claimed, err := s.store.ClaimNexusAlert(ctx, tenantID, st.StateCode, year, level, st.ProximityPct)
		if err != nil {
			s.logger.Error("claim nexus alert", "tenant_id", tenantID, "state", st.StateCode, "level", level, "error", err)
			continue
		}
		if !claimed {
			continue // already alerted this year at this level
		}

		// Resolve the recipient lazily — only once we actually have an alert to
		// send — and reuse it across states for the tenant.
		if !recipientLoaded {
			recipientEmail, recipientName = s.recipient(ctx, tenantID)
			recipientLoaded = true
		}
		if recipientEmail == "" {
			s.logger.Warn("nexus alert: no recipient for tenant; alert recorded but not sent", "tenant_id", tenantID, "state", st.StateCode)
			continue
		}

		data := email.NexusAlertData{
			RecipientEmail: recipientEmail,
			RecipientName:  recipientName,
			State:          st.StateCode,
			Level:          level,
			ProximityPct:   st.ProximityPct,
			TaxableSales:   formatUSDCents(st.TaxableSales),
			TxnCount:       st.TxnCount,
			ThresholdText:  thresholdText(st.Threshold),
			SettingsURL:    s.settingsURL,
		}
		if err := s.notifier.SendNexusThresholdAlert(ctx, level, data); err != nil {
			s.logger.Error("send nexus alert", "tenant_id", tenantID, "state", st.StateCode, "level", level, "error", err)
			continue
		}
		s.logger.Info("nexus threshold alert sent", "tenant_id", tenantID, "state", st.StateCode, "level", level, "proximity", st.ProximityPct)
	}
	return nil
}

// levelFor decides which alert (if any) a state's status warrants this year:
//   - "crossed": economic nexus was established this calendar year (the crossing
//     moment; established_at year is stamped by EstablishEconomic). Declared
//     physical/voluntary nexus is not alerted — the tenant set it themselves.
//   - "approaching": no nexus yet, but activity has reached the approaching band.
func levelFor(st domain.NexusStateStatus, year int) string {
	switch {
	case st.NexusType == domain.NexusEconomic && st.EstablishedAt != nil && st.EstablishedAt.Year() == year:
		return string(domain.NexusAlertCrossed)
	case st.NexusType == "" && st.ProximityPct >= nexusApproachingPct:
		return string(domain.NexusAlertApproaching)
	default:
		return ""
	}
}

// recipient picks the tenant's alert recipient: owner first, then any admin,
// then any user. Returns empty when the tenant has no users (or on error).
func (s *NexusAlertService) recipient(ctx context.Context, tenantID uuid.UUID) (emailAddr, name string) {
	users, err := s.users.ListByTenant(ctx, tenantID)
	if err != nil {
		s.logger.Error("nexus alert: list users", "tenant_id", tenantID, "error", err)
		return "", ""
	}
	var owner, admin, fallback *domain.User
	for _, u := range users {
		if fallback == nil {
			fallback = u
		}
		switch u.Role {
		case domain.RoleOwner:
			if owner == nil {
				owner = u
			}
		case domain.RoleAdmin:
			if admin == nil {
				admin = u
			}
		}
	}
	pick := owner
	if pick == nil {
		pick = admin
	}
	if pick == nil {
		pick = fallback
	}
	if pick == nil {
		return "", ""
	}
	return pick.Email, pick.Name
}

// thresholdText renders a state's economic-nexus threshold for display, e.g.
// "$100,000.00 or 200 transactions". Empty when the threshold is unknown.
func thresholdText(t *domain.NexusThreshold) string {
	if t == nil {
		return ""
	}
	var parts []string
	if t.SalesThreshold != nil {
		parts = append(parts, formatUSDCents(*t.SalesThreshold))
	}
	if t.TxnThreshold != nil {
		parts = append(parts, fmt.Sprintf("%d transactions", *t.TxnThreshold))
	}
	if len(parts) == 0 {
		return ""
	}
	sep := " or "
	if t.Combinator == "and" {
		sep = " and "
	}
	return strings.Join(parts, sep)
}

// formatUSDCents renders minor units (USD cents) as "$1,234.56".
func formatUSDCents(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	dollars := cents / 100
	frac := cents % 100

	// group the integer part with thousands separators
	digits := fmt.Sprintf("%d", dollars)
	var grouped strings.Builder
	n := len(digits)
	for i, d := range digits {
		if i > 0 && (n-i)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(d)
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s$%s.%02d", sign, grouped.String(), frac)
}
