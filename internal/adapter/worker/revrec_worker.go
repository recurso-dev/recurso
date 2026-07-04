package worker

import (
	"context"
	"log"
	"time"

	"github.com/swapnull-in/recur-so/internal/service"
)

type RevRecWorker struct {
	revrecService *service.RevRecService
	interval      time.Duration
}

func NewRevRecWorker(revrecService *service.RevRecService, interval time.Duration) *RevRecWorker {
	return &RevRecWorker{
		revrecService: revrecService,
		interval:      interval,
	}
}

func (w *RevRecWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("RevRec Worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunRecognition(ctx)
		}
	}
}

func (w *RevRecWorker) RunRecognition(ctx context.Context) {
	log.Println("Running Revenue Recognition processing...")
	if err := w.revrecService.ProcessDueEvents(ctx); err != nil {
		log.Printf("Failed to process revenue recognition: %v", err)
	}
}
