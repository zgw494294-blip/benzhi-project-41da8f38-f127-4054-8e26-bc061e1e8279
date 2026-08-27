package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/evidence"
)

func (t *transaction) InsertArtifact(a domain.ReleaseArtifact) error {
	_, err := t.tx.Exec(`INSERT INTO release_artifacts(id,job_id,revision_id,credential_number,webvtt,manifest,webvtt_digest,manifest_digest,audit_head_digest,frozen_by,frozen_at,verification_status) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, a.ID, a.JobID, a.RevisionID, a.CredentialNumber, a.WebVTT, a.Manifest, a.WebVTTDigest, a.ManifestDigest, a.AuditHeadDigest, a.FrozenBy, a.FrozenAt.Format(time.RFC3339Nano), a.VerificationStatus)
	return err
}

func (t *transaction) InsertVerification(v domain.VerificationRecord) error {
	b, _ := json.Marshal(v.Reasons)
	_, err := t.tx.Exec(`INSERT INTO verification_records(id,job_id,artifact_id,verifier_id,valid,reasons_json,webvtt_digest,manifest_digest,audit_head_digest,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, v.ID, v.JobID, v.ArtifactID, v.VerifierID, v.Valid, string(b), v.WebVTTDigest, v.ManifestDigest, v.AuditHeadDigest, v.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (t *transaction) ListVerifications(jobID string, limit, offset int) ([]domain.VerificationRecord, error) {
	rows, err := t.tx.Query(`SELECT id,job_id,artifact_id,verifier_id,valid,reasons_json,webvtt_digest,manifest_digest,audit_head_digest,created_at FROM verification_records WHERE job_id=? ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?`, jobID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.VerificationRecord{}
	for rows.Next() {
		var v domain.VerificationRecord
		var valid int
		var reasons, at string
		if err := rows.Scan(&v.ID, &v.JobID, &v.ArtifactID, &v.VerifierID, &valid, &reasons, &v.WebVTTDigest, &v.ManifestDigest, &v.AuditHeadDigest, &at); err != nil {
			return nil, err
		}
		v.Valid = valid != 0
		_ = json.Unmarshal([]byte(reasons), &v.Reasons)
		v.CreatedAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (t *transaction) GetArtifactByJob(jobID string) (domain.ReleaseArtifact, error) {
	var a domain.ReleaseArtifact
	var at string
	err := t.tx.QueryRow(`SELECT id,job_id,revision_id,credential_number,webvtt,manifest,webvtt_digest,manifest_digest,audit_head_digest,frozen_by,frozen_at,verification_status FROM release_artifacts WHERE job_id=?`, jobID).Scan(&a.ID, &a.JobID, &a.RevisionID, &a.CredentialNumber, &a.WebVTT, &a.Manifest, &a.WebVTTDigest, &a.ManifestDigest, &a.AuditHeadDigest, &a.FrozenBy, &at, &a.VerificationStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return a, domain.NewError(domain.CodeNotFound, "发布物不存在")
	}
	if err != nil {
		return a, err
	}
	a.FrozenAt, err = time.Parse(time.RFC3339Nano, at)
	return a, err
}
func (t *transaction) UpdateArtifactStatus(jobID, status string) error {
	_, err := t.tx.Exec(`UPDATE release_artifacts SET verification_status=? WHERE job_id=?`, status, jobID)
	return err
}

func (t *transaction) GetIdempotency(operation, key string) ([]byte, bool, error) {
	var data []byte
	err := t.tx.QueryRow(`SELECT result_json FROM idempotency_records WHERE operation=? AND idem_key=?`, operation, key).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return data, err == nil, err
}
func (t *transaction) PutIdempotency(operation, key string, data []byte) error {
	_, err := t.tx.Exec(`INSERT INTO idempotency_records(operation,idem_key,result_json) VALUES(?,?,?)`, operation, key, data)
	return err
}

func (t *transaction) LastAuditDigest(jobID string) (string, error) {
	var digest string
	err := t.tx.QueryRow(`SELECT digest FROM audit_events WHERE job_id=? ORDER BY id DESC LIMIT 1`, jobID).Scan(&digest)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return digest, err
}
func (t *transaction) AppendAudit(e domain.AuditEvent) error {
	_, err := t.tx.Exec(`INSERT INTO audit_events(job_id,event_type,actor_id,payload,prev_digest,digest,occurred_at) VALUES(?,?,?,?,?,?,?)`, e.JobID, e.EventType, e.ActorID, e.Payload, e.PrevDigest, e.Digest, e.OccurredAt.Format(time.RFC3339Nano))
	return err
}
func scanAudit(row scanner) (domain.AuditEvent, error) {
	var e domain.AuditEvent
	var at string
	err := row.Scan(&e.ID, &e.JobID, &e.EventType, &e.ActorID, &e.Payload, &e.PrevDigest, &e.Digest, &at)
	if err != nil {
		return e, err
	}
	e.OccurredAt, err = time.Parse(time.RFC3339Nano, at)
	return e, err
}
func (t *transaction) ListAudit(jobID string, limit, offset int) ([]domain.AuditEvent, error) {
	rows, err := t.tx.Query(`SELECT id,job_id,event_type,actor_id,payload,prev_digest,digest,occurred_at FROM audit_events WHERE job_id=? ORDER BY id LIMIT ? OFFSET ?`, jobID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.AuditEvent{}
	for rows.Next() {
		e, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
func (t *transaction) verifyAudit(jobID string) error {
	events, err := t.ListAudit(jobID, 1_000_000, 0)
	if err != nil {
		return err
	}
	if _, err := evidence.VerifyAudit(events); err != nil {
		return fmt.Errorf("任务 %s 的审计链无效: %w", jobID, err)
	}
	return nil
}
