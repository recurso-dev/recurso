package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// Usage Platform v1.
//
// Read paths over usage_events. The table has NO tenant_id column, so
// every read is tenant-scoped in the repository via a join on
// subscriptions.tenant_id (see adapter/db/usage_repository.go).
//
//   - QueryUsage:            time-windowed buckets (date_trunc day|month)
//   - GetSubscriptionUsage:  current billing period + lifetime, per
//     dimension, with the customer's entitlement limit joined in when a
//     feature_key equal to the dimension name exists
//   - ListDimensions:        the tenant's dimension catalog

// UsageError is a sentinel error category for usage flows.
type UsageError string

func (e UsageError) Error() string { return string(e) }

// ErrUsageSubscriptionNotFound maps to HTTP 404 (also returned for
// subscriptions belonging to another tenant, to avoid existence leaks).
var (
	ErrUsageSubscriptionNotFound = UsageError("subscription not found")
	ErrUsageCustomerMismatch     = UsageError("customer does not match subscription")
)

// UsageValidationError marks invalid caller input (maps to HTTP 400).
type UsageValidationError string

func (e UsageValidationError) Error() string { return string(e) }

// defaultUsageWindow is the query window when from/to are omitted.
const defaultUsageWindow = 30 * 24 * time.Hour

// Event property bounds (usage-based billing v1).
const (
	maxEventProperties       = 20
	maxEventPropertyKeyLen   = 100
	maxEventPropertyValueLen = 255
	// maxTransactionIDLen matches usage_events.transaction_id VARCHAR(255).
	maxTransactionIDLen = 255
)

// usageEntitlementChecker is the slice of EntitlementService the usage
// platform needs; *EntitlementService satisfies it.
type usageEntitlementChecker interface {
	CheckFeature(ctx context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error)
}

type UsageService struct {
	usage         port.UsageRepository
	subscriptions port.SubscriptionRepository
	entitlements  usageEntitlementChecker
	now           func() time.Time // injectable for tests
}

func NewUsageService(
	usage port.UsageRepository,
	subscriptions port.SubscriptionRepository,
	entitlements usageEntitlementChecker,
) *UsageService {
	return &UsageService{
		usage:         usage,
		subscriptions: subscriptions,
		entitlements:  entitlements,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

// RecordEvent persists one event; see RecordEventIdempotent for the flag.
func (s *UsageService) RecordEvent(ctx context.Context, tenantID uuid.UUID, event *domain.UsageEvent) error {
	_, err := s.RecordEventIdempotent(ctx, tenantID, event)
	return err
}

// RecordEventIdempotent persists a metered usage event (POST
// /v1/usage/events) after verifying the subscription belongs to the
// caller's tenant and the event's customer matches the subscription —
// otherwise any tenant could inflate another tenant's metered usage. When
// the caller supplies a transaction_id, a retry of the same event collapses
// to the original (duplicate=true, Lago-parity C1).
func (s *UsageService) RecordEventIdempotent(ctx context.Context, tenantID uuid.UUID, event *domain.UsageEvent) (bool, error) {
	if err := validateUsageEvent(event); err != nil {
		return false, err
	}

	sub, err := s.subscriptions.GetByID(ctx, event.SubscriptionID)
	if err != nil {
		return false, err
	}
	if sub == nil || sub.TenantID != tenantID {
		return false, ErrUsageSubscriptionNotFound
	}
	if sub.CustomerID != event.CustomerID {
		return false, ErrUsageCustomerMismatch
	}
	return s.usage.RecordEventIdempotent(ctx, event)
}

// validateUsageEvent enforces the per-event input bounds.
func validateUsageEvent(event *domain.UsageEvent) error {
	// A usage event records consumption; quantity must be positive. Without this
	// a negative quantity would offset legitimate metered usage at aggregation
	// time (SUM), underbilling the customer. (binding:"required" rejects 0 but
	// not negatives.)
	if event.Quantity <= 0 {
		return UsageValidationError("quantity must be greater than zero")
	}
	if len(event.TransactionID) > maxTransactionIDLen {
		return UsageValidationError(fmt.Sprintf("transaction_id must be at most %d characters", maxTransactionIDLen))
	}
	// Bound free-form properties so one caller can't bloat the events table
	// or the aggregation JSONB paths (usage-based billing v1).
	if len(event.Properties) > maxEventProperties {
		return UsageValidationError(fmt.Sprintf("at most %d properties per event", maxEventProperties))
	}
	for k, v := range event.Properties {
		if k == "" || len(k) > maxEventPropertyKeyLen {
			return UsageValidationError(fmt.Sprintf("property keys must be 1-%d characters", maxEventPropertyKeyLen))
		}
		if len(v) > maxEventPropertyValueLen {
			return UsageValidationError(fmt.Sprintf("property values must be at most %d characters", maxEventPropertyValueLen))
		}
	}
	return nil
}

// maxUsageBatchSize bounds one batch-ingest request (Lago-parity C1).
const maxUsageBatchSize = 500

// BatchItemResult reports one event's outcome in a batch ingest.
type BatchItemResult struct {
	Index   int    `json:"index"`
	Status  string `json:"status"` // recorded | duplicate | error
	EventID string `json:"event_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// RecordEvents ingests up to maxUsageBatchSize events with per-item
// results — one bad event never fails the batch. Subscription ownership is
// verified once per distinct subscription.
func (s *UsageService) RecordEvents(ctx context.Context, tenantID uuid.UUID, events []*domain.UsageEvent) ([]BatchItemResult, error) {
	if len(events) == 0 {
		return nil, UsageValidationError("events must not be empty")
	}
	if len(events) > maxUsageBatchSize {
		return nil, UsageValidationError(fmt.Sprintf("at most %d events per batch", maxUsageBatchSize))
	}

	// One ownership check per distinct subscription in the batch.
	subs := map[uuid.UUID]*domain.Subscription{}
	results := make([]BatchItemResult, len(events))
	for i, event := range events {
		results[i].Index = i
		if err := validateUsageEvent(event); err != nil {
			results[i].Status, results[i].Error = "error", err.Error()
			continue
		}
		sub, ok := subs[event.SubscriptionID]
		if !ok {
			loaded, err := s.subscriptions.GetByID(ctx, event.SubscriptionID)
			if err != nil {
				results[i].Status, results[i].Error = "error", "subscription lookup failed"
				continue
			}
			sub = loaded
			subs[event.SubscriptionID] = sub
		}
		if sub == nil || sub.TenantID != tenantID {
			results[i].Status, results[i].Error = "error", ErrUsageSubscriptionNotFound.Error()
			continue
		}
		if sub.CustomerID != event.CustomerID {
			results[i].Status, results[i].Error = "error", ErrUsageCustomerMismatch.Error()
			continue
		}

		duplicate, err := s.usage.RecordEventIdempotent(ctx, event)
		switch {
		case err != nil:
			results[i].Status, results[i].Error = "error", "failed to record event"
		case duplicate:
			results[i].Status, results[i].EventID = "duplicate", event.ID.String()
		default:
			results[i].Status, results[i].EventID = "recorded", event.ID.String()
		}
	}
	return results, nil
}

// UsageQueryParams is the raw (pre-validation) input for QueryUsage.
type UsageQueryParams struct {
	SubscriptionID *uuid.UUID
	CustomerID     *uuid.UUID
	Dimension      string
	From           *time.Time
	To             *time.Time
	Granularity    string
}

// ResolvedUsageQuery echoes the effective window/granularity after
// defaulting, so callers can render what was actually queried.
type ResolvedUsageQuery struct {
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Granularity string    `json:"granularity"`
}

// QueryUsage validates/defaults the window and returns time-bucketed
// usage. Defaults: to=now, from=to-30d, granularity=day. At least one of
// subscription_id or customer_id is required.
func (s *UsageService) QueryUsage(ctx context.Context, tenantID uuid.UUID, params UsageQueryParams) ([]domain.UsageBucket, *ResolvedUsageQuery, error) {
	if params.SubscriptionID == nil && params.CustomerID == nil {
		return nil, nil, UsageValidationError("at least one of subscription_id or customer_id is required")
	}

	granularity := params.Granularity
	if granularity == "" {
		granularity = domain.UsageGranularityDay
	}
	if granularity != domain.UsageGranularityDay && granularity != domain.UsageGranularityMonth {
		return nil, nil, UsageValidationError(fmt.Sprintf("granularity must be %q or %q", domain.UsageGranularityDay, domain.UsageGranularityMonth))
	}

	to := s.now()
	if params.To != nil {
		to = params.To.UTC()
	}
	from := to.Add(-defaultUsageWindow)
	if params.From != nil {
		from = params.From.UTC()
	}
	if !from.Before(to) {
		return nil, nil, UsageValidationError("from must be before to")
	}

	buckets, err := s.usage.QueryUsage(ctx, tenantID, domain.UsageQueryFilter{
		SubscriptionID: params.SubscriptionID,
		CustomerID:     params.CustomerID,
		Dimension:      params.Dimension,
		From:           from,
		To:             to,
		Granularity:    granularity,
	})
	if err != nil {
		return nil, nil, err
	}
	if buckets == nil {
		buckets = []domain.UsageBucket{}
	}
	return buckets, &ResolvedUsageQuery{From: from, To: to, Granularity: granularity}, nil
}

// GetSubscriptionUsage reports the subscription's current billing period
// usage plus lifetime totals per dimension, joining in the customer's
// entitlement limit for feature_keys matching the dimension name — the
// "you've used 4,231 of 10,000 api_calls" view.
func (s *UsageService) GetSubscriptionUsage(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.SubscriptionUsage, error) {
	sub, err := s.subscriptions.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.TenantID != tenantID {
		return nil, ErrUsageSubscriptionNotFound
	}

	dims, err := s.usage.GetSubscriptionUsageByDimension(ctx, tenantID, subscriptionID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return nil, err
	}
	if dims == nil {
		dims = []domain.SubscriptionDimensionUsage{}
	}

	for i := range dims {
		limit, err := s.dimensionLimit(ctx, tenantID, sub.CustomerID, dims[i].Dimension)
		if err != nil {
			return nil, err
		}
		if limit != nil {
			remaining := *limit - dims[i].PeriodQuantity
			dims[i].LimitValue = limit
			dims[i].Remaining = &remaining
		}
	}

	return &domain.SubscriptionUsage{
		SubscriptionID:     sub.ID,
		CustomerID:         sub.CustomerID,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		Dimensions:         dims,
	}, nil
}

// dimensionLimit resolves the customer's entitlement limit for a
// feature_key equal to the dimension name, or nil when the dimension isn't
// a valid feature key or no limit grant exists. Reuses the entitlement
// engine's fast-path check (one indexed query per dimension).
func (s *UsageService) dimensionLimit(ctx context.Context, tenantID, customerID uuid.UUID, dimension string) (*int64, error) {
	// Dimensions are free-form (VARCHAR(50)); only consult the entitlement
	// engine for names that are valid feature keys.
	if dimension == "" || len(dimension) > maxFeatureKeyLen || !featureKeyRe.MatchString(dimension) {
		return nil, nil
	}
	check, err := s.entitlements.CheckFeature(ctx, tenantID, customerID, dimension)
	if err != nil {
		return nil, err
	}
	if check == nil {
		return nil, nil
	}
	return check.LimitValue, nil
}

// ListDimensions returns the tenant's usage-dimension catalog.
func (s *UsageService) ListDimensions(ctx context.Context, tenantID uuid.UUID) ([]domain.UsageDimension, error) {
	dims, err := s.usage.ListDimensions(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if dims == nil {
		dims = []domain.UsageDimension{}
	}
	return dims, nil
}
