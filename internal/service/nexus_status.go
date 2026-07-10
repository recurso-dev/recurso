package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// NexusStatusService computes per-state US economic-nexus status (ENG-16,
// FR-TAX-10 Phase 2): year-to-date taxable sales and transaction counts per
// buyer state, proximity to each state's threshold, and automatic
// establishment of economic nexus when a threshold is crossed.
//
// Every method takes the tenant as an explicit parameter — no reads depend on
// a tenant in ctx, so background callers are immune to the tenant-context bug
// class by construction.
type NexusStatusService struct {
	repo   *db.TaxNexusRepository
	logger *slog.Logger
}

func NewNexusStatusService(repo *db.TaxNexusRepository) *NexusStatusService {
	return &NexusStatusService{repo: repo, logger: slog.Default()}
}

// DatasetCertified reports whether every threshold row has passed
// professional review. Until then callers must surface the caveat: the seed
// encodes commonly-cited values and is NOT filing-grade.
func (s *NexusStatusService) DatasetCertified(ctx context.Context) (bool, error) {
	ths, err := s.repo.ListThresholds(ctx)
	if err != nil {
		return false, err
	}
	for _, t := range ths {
		if !t.Certified {
			return false, nil
		}
	}
	return len(ths) > 0, nil
}

// Status returns the per-state nexus picture for the tenant's calendar year:
// every state where the tenant has declared/economic nexus or any US sales
// activity. Crossings found while computing are auto-established (idempotent),
// so viewing status never under-reports an obligation it just detected.
func (s *NexusStatusService) Status(ctx context.Context, tenantID uuid.UUID, year int) ([]domain.NexusStateStatus, error) {
	if _, err := s.EvaluateEconomicNexus(ctx, tenantID, year); err != nil {
		// Evaluation failure shouldn't block the read view.
		s.logger.Error("economic-nexus evaluation failed", "tenant_id", tenantID, "error", err)
	}

	nexus, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sales, err := s.repo.SalesByState(ctx, tenantID, year)
	if err != nil {
		return nil, err
	}
	thresholds, err := s.repo.ListThresholds(ctx)
	if err != nil {
		return nil, err
	}

	thByState := make(map[string]domain.NexusThreshold, len(thresholds))
	for _, t := range thresholds {
		thByState[t.StateCode] = t
	}

	byState := map[string]*domain.NexusStateStatus{}
	get := func(state string) *domain.NexusStateStatus {
		if st, ok := byState[state]; ok {
			return st
		}
		st := &domain.NexusStateStatus{StateCode: state}
		if t, ok := thByState[state]; ok {
			th := t
			st.Threshold = &th
		}
		byState[state] = st
		return st
	}

	for _, n := range nexus {
		st := get(n.StateCode)
		st.NexusType = n.NexusType
		st.EstablishedAt = n.EstablishedAt
	}
	for _, sl := range sales {
		st := get(sl.StateCode)
		st.TaxableSales = sl.TaxableSales
		st.TxnCount = sl.TxnCount
	}

	out := make([]domain.NexusStateStatus, 0, len(byState))
	for _, st := range byState {
		st.ProximityPct, st.Crossed = proximity(st.Threshold, st.TaxableSales, st.TxnCount)
		out = append(out, *st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StateCode < out[j].StateCode })
	return out, nil
}

// EvaluateEconomicNexus establishes economic nexus for every state whose
// threshold the tenant's year-to-date activity crosses and that has no nexus
// row yet. Returns the newly established state codes. Idempotent.
func (s *NexusStatusService) EvaluateEconomicNexus(ctx context.Context, tenantID uuid.UUID, year int) ([]string, error) {
	sales, err := s.repo.SalesByState(ctx, tenantID, year)
	if err != nil {
		return nil, err
	}
	if len(sales) == 0 {
		return nil, nil
	}
	thresholds, err := s.repo.ListThresholds(ctx)
	if err != nil {
		return nil, err
	}
	thByState := make(map[string]domain.NexusThreshold, len(thresholds))
	for _, t := range thresholds {
		thByState[t.StateCode] = t
	}

	var established []string
	now := time.Now().UTC()
	for _, sl := range sales {
		t, ok := thByState[sl.StateCode]
		if !ok {
			continue // no state sales tax, or not a known state code
		}
		th := t
		if _, crossed := proximity(&th, sl.TaxableSales, sl.TxnCount); !crossed {
			continue
		}
		isNew, err := s.repo.EstablishEconomic(ctx, tenantID, sl.StateCode, now)
		if err != nil {
			return established, err
		}
		if isNew {
			s.logger.Info("economic nexus established",
				"tenant_id", tenantID, "state", sl.StateCode,
				"taxable_sales", sl.TaxableSales, "txn_count", sl.TxnCount)
			established = append(established, sl.StateCode)
		}
	}
	return established, nil
}

// proximity computes how close activity is to a threshold (percent, capped at
// 999) and whether it is crossed. "or" states cross on either limb, so
// proximity is the max ratio; "and" states need both, so it's the min.
func proximity(t *domain.NexusThreshold, sales int64, txns int) (int, bool) {
	if t == nil {
		return 0, false
	}
	ratio := func(num, den int64) int {
		if den <= 0 {
			return 0
		}
		p := num * 100 / den
		if p > 999 {
			return 999
		}
		return int(p)
	}
	var limbs []int
	if t.SalesThreshold != nil {
		limbs = append(limbs, ratio(sales, *t.SalesThreshold))
	}
	if t.TxnThreshold != nil {
		limbs = append(limbs, ratio(int64(txns), int64(*t.TxnThreshold)))
	}
	if len(limbs) == 0 {
		return 0, false
	}
	pct := limbs[0]
	for _, l := range limbs[1:] {
		if t.Combinator == "and" {
			if l < pct {
				pct = l
			}
		} else if l > pct {
			pct = l
		}
	}
	return pct, pct >= 100
}
