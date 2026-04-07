package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPendingUploadRepository is a mock implementation of PendingUploadRepository
type MockPendingUploadRepository struct {
	mock.Mock
}

func (m *MockPendingUploadRepository) Create(ctx context.Context, upload *repository.PendingUpload) error {
	args := m.Called(ctx, upload)
	return args.Error(0)
}

func (m *MockPendingUploadRepository) GetByToken(ctx context.Context, token vo.UploadToken) (*repository.PendingUpload, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PendingUpload), args.Error(1)
}

func (m *MockPendingUploadRepository) GetByPhotoID(ctx context.Context, photoID vo.PhotoID) (*repository.PendingUpload, error) {
	args := m.Called(ctx, photoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PendingUpload), args.Error(1)
}

func (m *MockPendingUploadRepository) MarkAsCompleted(ctx context.Context, token vo.UploadToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockPendingUploadRepository) MarkAsExpired(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPendingUploadRepository) Delete(ctx context.Context, token vo.UploadToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockPendingUploadRepository) DeleteByPhotoID(ctx context.Context, photoID vo.PhotoID) error {
	args := m.Called(ctx, photoID)
	return args.Error(0)
}

func (m *MockPendingUploadRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPendingUploadRepository) CountActiveByAPIKey(ctx context.Context, apiKeyID string) (int64, error) {
	args := m.Called(ctx, apiKeyID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPendingUploadRepository) GetExpired(ctx context.Context, before time.Time) ([]*repository.PendingUpload, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.PendingUpload), args.Error(1)
}

func (m *MockPendingUploadRepository) ExpireTokensByPhotoID(ctx context.Context, photoID vo.PhotoID) error {
	args := m.Called(ctx, photoID)
	return args.Error(0)
}

// MockLogger is a mock implementation of the Logger interface
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(msg string, ctx ...map[string]interface{}) {
	m.Called(msg, ctx)
}

func (m *MockLogger) Error(msg string, err error, ctx ...map[string]interface{}) {
	m.Called(msg, err, ctx)
}

func (m *MockLogger) Warn(msg string, ctx ...map[string]interface{}) {
	m.Called(msg, ctx)
}

func (m *MockLogger) Debug(msg string, ctx ...map[string]interface{}) {
	m.Called(msg, ctx)
}

func TestNewCleanupService(t *testing.T) {
	mockRepo := new(MockPendingUploadRepository)
	mockLogger := new(MockLogger)

	tests := []struct {
		name            string
		interval        time.Duration
		retentionPeriod time.Duration
		wantInterval    time.Duration
		wantRetention   time.Duration
	}{
		{
			name:            "with custom values",
			interval:        10 * time.Minute,
			retentionPeriod: 48 * time.Hour,
			wantInterval:    10 * time.Minute,
			wantRetention:   48 * time.Hour,
		},
		{
			name:            "with zero values uses defaults",
			interval:        0,
			retentionPeriod: 0,
			wantInterval:    model.DefaultCleanupInterval,
			wantRetention:   model.DefaultCleanupRetention,
		},
		{
			name:            "with negative values uses defaults",
			interval:        -5 * time.Minute,
			retentionPeriod: -1 * time.Hour,
			wantInterval:    model.DefaultCleanupInterval,
			wantRetention:   model.DefaultCleanupRetention,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewCleanupService(mockRepo, mockLogger, tt.interval, tt.retentionPeriod)
			assert.Equal(t, tt.wantInterval, svc.interval)
			assert.Equal(t, tt.wantRetention, svc.retentionPeriod)
		})
	}
}

func TestCleanupService_runCleanup(t *testing.T) {
	t.Run("successful cleanup cycle", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		// Expect MarkAsExpired to be called with a time around 15 minutes ago
		mockRepo.On("MarkAsExpired", mock.Anything, mock.MatchedBy(func(before time.Time) bool {
			// Check that the cutoff is approximately 15 minutes ago
			diff := time.Since(before)
			return diff >= 14*time.Minute && diff <= 16*time.Minute
		})).Return(int64(5), nil)

		// Expect DeleteExpired to be called with a time around 24 hours ago
		mockRepo.On("DeleteExpired", mock.Anything, mock.MatchedBy(func(before time.Time) bool {
			// Check that the cutoff is approximately 24 hours ago
			diff := time.Since(before)
			return diff >= 23*time.Hour && diff <= 25*time.Hour
		})).Return(int64(3), nil)

		svc := NewCleanupService(mockRepo, mockLogger, 5*time.Minute, 24*time.Hour)
		marked, deleted, err := svc.runCleanup(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(5), marked)
		assert.Equal(t, int64(3), deleted)
		mockRepo.AssertExpectations(t)
	})

	t.Run("mark as expired fails", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		mockRepo.On("MarkAsExpired", mock.Anything, mock.Anything).Return(int64(0), errors.New("db error"))

		svc := NewCleanupService(mockRepo, mockLogger, 5*time.Minute, 24*time.Hour)
		marked, deleted, err := svc.runCleanup(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark uploads as expired")
		assert.Equal(t, int64(0), marked)
		assert.Equal(t, int64(0), deleted)
		mockRepo.AssertExpectations(t)
	})

	t.Run("delete expired fails but returns marked count", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		mockRepo.On("MarkAsExpired", mock.Anything, mock.Anything).Return(int64(5), nil)
		mockRepo.On("DeleteExpired", mock.Anything, mock.Anything).Return(int64(0), errors.New("db error"))

		svc := NewCleanupService(mockRepo, mockLogger, 5*time.Minute, 24*time.Hour)
		marked, deleted, err := svc.runCleanup(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete expired uploads")
		assert.Equal(t, int64(5), marked)
		assert.Equal(t, int64(0), deleted)
		mockRepo.AssertExpectations(t)
	})
}

func TestCleanupService_StartStop(t *testing.T) {
	t.Run("start and stop gracefully", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		// Allow multiple calls since service runs immediately on start + on ticks
		mockRepo.On("MarkAsExpired", mock.Anything, mock.Anything).Return(int64(0), nil)
		mockRepo.On("DeleteExpired", mock.Anything, mock.Anything).Return(int64(0), nil)
		mockLogger.On("Info", "Cleanup service started", mock.Anything).Once()
		mockLogger.On("Info", "Cleanup service stopped", mock.Anything).Once()

		svc := NewCleanupService(mockRepo, mockLogger, 100*time.Millisecond, 24*time.Hour)
		ctx, cancel := context.WithCancel(context.Background())

		svc.Start(ctx)

		// Let it run for a bit
		time.Sleep(150 * time.Millisecond)

		// Stop should complete without hanging
		svc.Stop()
		cancel()

		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("cleanup runs on each tick", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		// Expect at least 2 cleanup cycles (immediate + tick)
		mockRepo.On("MarkAsExpired", mock.Anything, mock.Anything).Return(int64(0), nil)
		mockRepo.On("DeleteExpired", mock.Anything, mock.Anything).Return(int64(0), nil)
		mockLogger.On("Info", "Cleanup service started", mock.Anything).Once()
		mockLogger.On("Info", "Cleanup service stopped", mock.Anything).Once()

		svc := NewCleanupService(mockRepo, mockLogger, 50*time.Millisecond, 24*time.Hour)
		ctx, cancel := context.WithCancel(context.Background())

		svc.Start(ctx)

		// Wait for at least 2 cleanup cycles (immediate + 1 tick)
		time.Sleep(150 * time.Millisecond)

		svc.Stop()
		cancel()

		mockRepo.AssertExpectations(t)
	})
}

func TestCleanupService_performCleanup_LogsResults(t *testing.T) {
	t.Run("logs results when work is done", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		mockRepo.On("MarkAsExpired", mock.Anything, mock.Anything).Return(int64(5), nil)
		mockRepo.On("DeleteExpired", mock.Anything, mock.Anything).Return(int64(3), nil)
		mockLogger.On("Info", "Cleanup cycle completed", mock.MatchedBy(func(ctx []map[string]interface{}) bool {
			if len(ctx) == 0 {
				return false
			}
			return ctx[0]["marked_expired"] == int64(5) && ctx[0]["deleted_expired"] == int64(3)
		})).Once()

		svc := NewCleanupService(mockRepo, mockLogger, 5*time.Minute, 24*time.Hour)
		svc.performCleanup(context.Background())

		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("logs error on failure", func(t *testing.T) {
		mockRepo := new(MockPendingUploadRepository)
		mockLogger := new(MockLogger)

		dbErr := errors.New("database connection lost")
		mockRepo.On("MarkAsExpired", mock.Anything, mock.Anything).Return(int64(0), dbErr)
		mockLogger.On("Error", "Cleanup cycle failed", mock.Anything, mock.Anything).Once()

		svc := NewCleanupService(mockRepo, mockLogger, 5*time.Minute, 24*time.Hour)
		svc.performCleanup(context.Background())

		mockRepo.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})
}
