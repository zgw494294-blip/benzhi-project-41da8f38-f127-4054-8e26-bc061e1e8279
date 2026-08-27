package sqlite

import "context"

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS caption_jobs (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, media_ref TEXT NOT NULL, duration_millis INTEGER NOT NULL,
			source_language TEXT NOT NULL, target_language TEXT NOT NULL, profile_code TEXT NOT NULL,
			status TEXT NOT NULL, current_revision_id TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS caption_revisions (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES caption_jobs(id), parent_revision_id TEXT NOT NULL DEFAULT '',
			sequence INTEGER NOT NULL, author_id TEXT NOT NULL, change_note TEXT NOT NULL, segments_json TEXT NOT NULL,
			content_digest TEXT NOT NULL, submitted_at TEXT NOT NULL, UNIQUE(job_id, sequence))`,
		`CREATE INDEX IF NOT EXISTS idx_revisions_job ON caption_revisions(job_id, sequence)`,
		`CREATE TABLE IF NOT EXISTS quality_findings (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES caption_jobs(id), revision_id TEXT NOT NULL REFERENCES caption_revisions(id),
			rule_code TEXT NOT NULL, severity TEXT NOT NULL, segment_ids_json TEXT NOT NULL, message TEXT NOT NULL,
			disposition TEXT NOT NULL, rationale TEXT NOT NULL DEFAULT '', resolved_by TEXT NOT NULL DEFAULT '', resolved_at TEXT)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_revision ON quality_findings(job_id, revision_id)`,
		`CREATE TABLE IF NOT EXISTS quality_check_batches (id TEXT PRIMARY KEY, job_id TEXT NOT NULL, revision_id TEXT NOT NULL, profile_code TEXT NOT NULL, rule_version TEXT NOT NULL, snapshot_json TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS review_decisions (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES caption_jobs(id), revision_id TEXT NOT NULL REFERENCES caption_revisions(id),
			reviewer_id TEXT NOT NULL, sampled_segment_ids_json TEXT NOT NULL, decision TEXT NOT NULL, comment TEXT NOT NULL, decided_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS release_artifacts (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL UNIQUE REFERENCES caption_jobs(id), revision_id TEXT NOT NULL REFERENCES caption_revisions(id),
			credential_number TEXT NOT NULL UNIQUE, webvtt TEXT NOT NULL, manifest TEXT NOT NULL, webvtt_digest TEXT NOT NULL,
			manifest_digest TEXT NOT NULL, audit_head_digest TEXT NOT NULL, frozen_by TEXT NOT NULL, frozen_at TEXT NOT NULL, verification_status TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS idempotency_records (
			operation TEXT NOT NULL, idem_key TEXT NOT NULL, result_json BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY(operation, idem_key))`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT, job_id TEXT NOT NULL REFERENCES caption_jobs(id), event_type TEXT NOT NULL,
			actor_id TEXT NOT NULL, payload TEXT NOT NULL, prev_digest TEXT NOT NULL, digest TEXT NOT NULL UNIQUE, occurred_at TEXT NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_job ON audit_events(job_id, id)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_queue ON caption_jobs(status, profile_code, source_language, target_language, updated_at DESC, id)`,
		`CREATE TABLE IF NOT EXISTS verification_records (
			id TEXT PRIMARY KEY, job_id TEXT NOT NULL REFERENCES caption_jobs(id), artifact_id TEXT NOT NULL,
			verifier_id TEXT NOT NULL, valid INTEGER NOT NULL, reasons_json TEXT NOT NULL,
			webvtt_digest TEXT NOT NULL, manifest_digest TEXT NOT NULL, audit_head_digest TEXT NOT NULL,
			created_at TEXT NOT NULL)
		`,
		`CREATE INDEX IF NOT EXISTS idx_verification_job ON verification_records(job_id, created_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
