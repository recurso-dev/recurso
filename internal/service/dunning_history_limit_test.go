package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// recordingHistoryRepo captures the limit the service forwards.
type recordingHistoryRepo struct {
	stubTimingRepo
	gotLimit int
}

func (r *recordingHistoryRepo) GetRecentHistory(_ context.Context, _ uuid.UUID, limit int) ([]domain.DunningHistory, error) {
	r.gotLimit = limit
	return nil, nil
}

// A limit above the cap must be capped, not reset to the default: the handler
// already caps at 500, and a caller asking for 300 used to silently get 50.
func TestDunningAnalytics_GetRecentHistory_ClampsLikeTheHandler(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, 50}, {-5, 50}, {1, 1}, {200, 200}, {300, 300}, {500, 500}, {501, 500}, {100000, 500},
	}
	for _, tc := range cases {
		repo := &recordingHistoryRepo{}
		svc := NewDunningAnalyticsService(repo)
		if _, err := svc.GetRecentHistory(context.Background(), uuid.New(), tc.in); err != nil {
			t.Fatalf("limit %d: %v", tc.in, err)
		}
		if repo.gotLimit != tc.want {
			t.Errorf("limit %d forwarded as %d, want %d", tc.in, repo.gotLimit, tc.want)
		}
	}
}
