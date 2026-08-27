package application

import (
	"context"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

type Repository interface {
	Transact(context.Context, func(Tx) error) error
	Close() error
	CheckIntegrity(context.Context) error
}

type Tx interface {
	GetJob(string) (domain.CaptionJob, error)
	ListJobs(limit, offset int) ([]domain.CaptionJob, error)
	ListJobsFiltered(filter JobQueueFilter) ([]domain.CaptionJob, int, map[domain.JobStatus]int, error)
	InsertJob(domain.CaptionJob) error
	UpdateJob(domain.CaptionJob, int64) error
	GetRevision(string) (domain.CaptionRevision, error)
	ListRevisions(string) ([]domain.CaptionRevision, error)
	InsertRevision(domain.CaptionRevision) error
	GetFinding(string) (domain.QualityFinding, error)
	ListFindings(string, string) ([]domain.QualityFinding, error)
	InsertFindings([]domain.QualityFinding) error
	UpdateFinding(domain.QualityFinding) error
	UpdateFindings([]domain.QualityFinding) error
	InsertReview(domain.ReviewDecision) error
	ListReviews(string) ([]domain.ReviewDecision, error)
	InsertArtifact(domain.ReleaseArtifact) error
	GetArtifactByJob(string) (domain.ReleaseArtifact, error)
	UpdateArtifactStatus(string, string) error
	ListAudit(string, int, int) ([]domain.AuditEvent, error)
	LastAuditDigest(string) (string, error)
	AppendAudit(domain.AuditEvent) error
	GetIdempotency(string, string) ([]byte, bool, error)
	PutIdempotency(string, string, []byte) error
	InsertVerification(domain.VerificationRecord) error
	ListVerifications(string, int, int) ([]domain.VerificationRecord, error)
	RecordQualitySnapshot(string, string, string, string, string, []byte, string) error
}
