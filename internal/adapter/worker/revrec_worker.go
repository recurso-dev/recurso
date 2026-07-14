package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/recurso-dev/recurso/internal/service"
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

	slog.Info("revrec worker started")

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
	slog.Info("running revenue recognition processing")
	if err := w.revrecService.ProcessDueEvents(ctx); err != nil {
		slog.Error("failed to process revenue recognition", "error", err)
	}
}
