package worker

import (
	"context"
	"log"
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

	log.Println("DunningCampaignWorker started...")

	for {
		select {
		case <-ctx.Done():
			log.Println("DunningCampaignWorker stopping...")
			return
		case <-ticker.C:
			if err := w.campaignService.ProcessDueSteps(ctx); err != nil {
				log.Printf("DunningCampaignWorker: error processing due steps: %v", err)
			}
		}
	}
}
