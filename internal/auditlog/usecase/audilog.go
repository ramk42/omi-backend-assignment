package usecase

import (
	"context"
	"errors"
	"github.com/ramk42/omi-backend-assignment/internal/auditlog"
	"github.com/rs/zerolog/log"
	"sync"
	"sync/atomic"
	"time"
)

type AuditLog struct {
	repository    auditlog.Repository
	buffer        []*auditlog.Model
	mu            sync.Mutex
	ctx           context.Context
	wg            sync.WaitGroup
	once          sync.Once
	isClosed      atomic.Bool
	flushChan     chan struct{}
	flushInterval time.Duration
	bufferSize    int
}

func NewAuditLog(
	ctx context.Context,
	auditLogRepo auditlog.Repository,
	bufferSize int,
	flushInterval time.Duration,
) *AuditLog {

	al := &AuditLog{
		repository:    auditLogRepo,
		buffer:        make([]*auditlog.Model, 0, bufferSize),
		flushChan:     make(chan struct{}, 1),
		flushInterval: flushInterval,
		bufferSize:    bufferSize,
	}

	al.wg.Add(1)
	go al.batchProcessor(ctx)

	return al
}

func (a *AuditLog) Push(ctx context.Context, model *auditlog.Model) error {
	if a.isClosed.Load() {
		log.Warn().Msg("audit log closed: refusing new entry")
		return auditlog.ErrAuditLogClosed
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.buffer) >= a.bufferSize {
		log.Warn().Msg("buffer full: triggering emergency flush")
		a.flushChan <- struct{}{}
		return auditlog.ErrAuditLogBufferFull
	}

	a.buffer = append(a.buffer, model)

	if len(a.buffer) == a.bufferSize {
		log.Debug().Msg("buffer full: triggering flush")
		a.flushChan <- struct{}{}
	}

	return nil
}

func (a *AuditLog) batchProcessor(ctx context.Context) {
	defer a.wg.Done()
	ticker := time.NewTicker(a.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.safeFlush(ctx)
			return
		case <-ticker.C:
			log.Info().Msg("batch processor triggered")
			a.safeFlush(ctx)
		case <-a.flushChan:
			a.safeFlush(ctx)
		}
	}
}

func (a *AuditLog) safeFlush(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.buffer) == 0 {
		log.Debug().Msg("nothing to flush")
		return
	}

	batchCopy := make([]*auditlog.Model, len(a.buffer))
	copy(batchCopy, a.buffer)
	a.buffer = a.buffer[:0] // reset buffer
	if err := a.repository.Insert(ctx, batchCopy); err != nil {
		log.Error().Err(err).Msg("failed to flush audit log")
		if errors.Is(err, auditlog.ErrDatabaseUnavailable) {
			go a.retryUntilDBAvailable(ctx, batchCopy)
		}
		return
	}
}

func (a *AuditLog) retryUntilDBAvailable(ctx context.Context, batch []*auditlog.Model) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Info().Msg("retrying until database is available")
			err := a.repository.Insert(ctx, batch)
			if err == nil {
				log.Info().Int("count", len(batch)).Msg("audit logs successfully flushed after database recovery")
				return
			}
			log.Info().Err(err).Msg("database still unavailable, retrying...")

		case <-ctx.Done():
			log.Warn().Msg("shutting down, aborting retry")
			return
		}
	}
}

func (a *AuditLog) Close() {
	a.once.Do(func() {
		a.isClosed.Store(true)
		log.Info().Msg("initiating usecase shutdown...")
		a.wg.Wait()
		a.mu.Lock()
		defer a.mu.Unlock()
		a.buffer = nil
	})
}
