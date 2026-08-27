package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/evidence"
)

type Service struct {
	repo           Repository
	clock          func() time.Time
	preflightMu    sync.RWMutex
	preflightByJob map[string]RevisionPreflight
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:           repo,
		clock:          func() time.Time { return time.Now().UTC() },
		preflightByJob: make(map[string]RevisionPreflight),
	}
}

func newID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b)
}

func validateKey(key string) error {
	if len(strings.TrimSpace(key)) < 8 || len(key) > 128 {
		return domain.NewError(domain.CodeValidation, "idempotencyKey 长度必须在 8 到 128 之间")
	}
	return nil
}

func expectVersion(job domain.CaptionJob, expected int64) error {
	if job.Version != expected {
		return domain.NewError(domain.CodeConflict, "版本冲突：当前版本为 %d", job.Version)
	}
	return nil
}

func appendAudit(tx Tx, jobID, eventType, actor string, payload any, at time.Time) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	previous, err := tx.LastAuditDigest(jobID)
	if err != nil {
		return err
	}
	event := domain.AuditEvent{JobID: jobID, EventType: eventType, ActorID: actor, Payload: string(data), PrevDigest: previous, OccurredAt: at}
	event.Digest = evidence.AuditDigest(previous, jobID, eventType, actor, event.Payload, at)
	return tx.AppendAudit(event)
}

func idemLoad[T any](tx Tx, operation, key string) (T, bool, error) {
	var zero T
	data, found, err := tx.GetIdempotency(operation, key)
	if err != nil || !found {
		return zero, found, err
	}
	if err := json.Unmarshal(data, &zero); err != nil {
		return zero, false, fmt.Errorf("读取幂等结果: %w", err)
	}
	return zero, true, nil
}

func idemSave(tx Tx, operation, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.PutIdempotency(operation, key, data)
}

func (s *Service) CreateJob(ctx context.Context, cmd CreateJobCommand) (domain.CaptionJob, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return domain.CaptionJob{}, err
	}
	var result domain.CaptionJob
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if saved, ok, err := idemLoad[domain.CaptionJob](tx, "create_job", cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			result = saved
			return nil
		}
		now := s.clock()
		job := domain.CaptionJob{ID: newID("job"), Title: strings.TrimSpace(cmd.Title), MediaRef: strings.TrimSpace(cmd.MediaRef), DurationMillis: cmd.DurationMillis, SourceLanguage: strings.TrimSpace(cmd.SourceLanguage), TargetLanguage: strings.TrimSpace(cmd.TargetLanguage), ProfileCode: strings.TrimSpace(cmd.ProfileCode), Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
		if err := domain.ValidateNewJob(job); err != nil {
			return err
		}
		if strings.TrimSpace(cmd.ActorID) == "" {
			return domain.NewError(domain.CodeValidation, "建档人不能为空")
		}
		if err := tx.InsertJob(job); err != nil {
			return err
		}
		if err := appendAudit(tx, job.ID, "job.created", cmd.ActorID, map[string]any{"title": job.Title, "version": job.Version}, now); err != nil {
			return err
		}
		if err := idemSave(tx, "create_job", cmd.IdempotencyKey, job); err != nil {
			return err
		}
		result = job
		return nil
	})
	return result, err
}
