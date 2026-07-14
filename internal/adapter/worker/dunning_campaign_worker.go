package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/recurso-dev/recurso/internal/service"
)

// DunningCampaignWorker polls for due campaign steps and processes them
type DunningCampaignWorker struct {
	campaignService *service.DunningCampaignService
	interval        time.Duration
}

func NewDunningCampaignWorker(campaignService *service.DunningCampaignService) *DunningCampaignWorker {
	return &DunningCampaignWorker{
		campaignService: campaignService,
		interval:        60 * time.Second,
	}
}

func (w *DunningCampaignWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("dunning campaign worker started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("dunning campaign worker stopping")
			return
		case <-ticker.C:
			if err := w.campaignService.ProcessDueSteps(ctx); err != nil {
				slog.Error("dunning campaign worker: failed to process due steps", "error", err)
			}
		}
	}
}
