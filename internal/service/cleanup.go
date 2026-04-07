// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// CleanupService periodically cleans up expired pending uploads.
// It runs a two-phase cleanup process:
// 1. Marks pending uploads as expired when their tokens are past expiry
// 2. Deletes old expired records after a retention period
type CleanupService struct {
	pendingUploadRepo repository.PendingUploadRepository
	logger            Logger
	interval          time.Duration
	retentionPeriod   time.Duration
	stopCh            chan struct{}
	doneCh            chan struct{}
}

// NewCleanupService creates a new cleanup service.
func NewCleanupService(
	pendingUploadRepo repository.PendingUploadRepository,
	logger Logger,
	interval time.Duration,
	retentionPeriod time.Duration,
) *CleanupService {
	if interval <= 0 {
		interval = model.DefaultCleanupInterval
	}
	if retentionPeriod <= 0 {
		retentionPeriod = model.DefaultCleanupRetention
	}

	return &CleanupService{
		pendingUploadRepo: pendingUploadRepo,
		logger:            logger,
		interval:          interval,
		retentionPeriod:   retentionPeriod,
		stopCh:            make(chan struct{}),
		doneCh:            make(chan struct{}),
	}
}

// Start begins the periodic cleanup loop in a goroutine.
func (s *CleanupService) Start(ctx context.Context) {
	go s.run(ctx)
}

// Stop gracefully stops the cleanup loop.
func (s *CleanupService) Stop() {
	close(s.stopCh)
	<-s.doneCh
}

// run is the main cleanup loop.
func (s *CleanupService) run(ctx context.Context) {
	defer close(s.doneCh)

	// Run immediately on start
	s.logger.Info("Cleanup service started", map[string]interface{}{
		"interval":         s.interval,
		"retention_period": s.retentionPeriod,
	})
	s.performCleanup(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Cleanup service stopped due to context cancellation")
			return
		case <-s.stopCh:
			s.logger.Info("Cleanup service stopped")
			return
		case <-ticker.C:
			s.performCleanup(ctx)
		}
	}
}

// performCleanup executes one cleanup cycle.
func (s *CleanupService) performCleanup(ctx context.Context) {
	marked, deleted, err := s.runCleanup(ctx)
	if err != nil {
		s.logger.Error("Cleanup cycle failed", err, map[string]interface{}{
			"marked_count":  marked,
			"deleted_count": deleted,
		})
		return
	}

	if marked > 0 || deleted > 0 {
		s.logger.Info("Cleanup cycle completed", map[string]interface{}{
			"marked_expired":  marked,
			"deleted_expired": deleted,
		})
	}
}

// runCleanup executes the two-phase cleanup and returns counts.
func (s *CleanupService) runCleanup(ctx context.Context) (marked int64, deleted int64, err error) {
	now := time.Now()

	// Phase 1: Mark pending uploads as expired if their expiry time has passed
	// We use a 15-minute buffer to allow clients to get proper error messages
	expiredCutoff := now.Add(-model.UploadTokenExpiry)
	marked, err = s.pendingUploadRepo.MarkAsExpired(ctx, expiredCutoff)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to mark uploads as expired: %w", err)
	}

	// Phase 2: Delete old expired records that are past the retention period
	retentionCutoff := now.Add(-s.retentionPeriod)
	deleted, err = s.pendingUploadRepo.DeleteExpired(ctx, retentionCutoff)
	if err != nil {
		return marked, 0, fmt.Errorf("failed to delete expired uploads: %w", err)
	}

	return marked, deleted, nil
}
