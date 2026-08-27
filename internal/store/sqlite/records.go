package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

func (t *transaction) RecordQualitySnapshot(id, jobID, revisionID, profile, version string, snapshot []byte, created string) error {
	_, err := t.tx.Exec(`INSERT INTO quality_check_batches(id,job_id,revision_id,profile_code,rule_version,snapshot_json,created_at) VALUES(?,?,?,?,?,?,?)`, id, jobID, revisionID, profile, version, string(snapshot), created)
	return err
}

func (t *transaction) InsertRevision(r domain.CaptionRevision) error {
	data, err := json.Marshal(r.Segments)
	if err != nil {
		return err
	}
	_, err = t.tx.Exec(`INSERT INTO caption_revisions(id,job_id,parent_revision_id,sequence,author_id,change_note,segments_json,content_digest,submitted_at) VALUES(?,?,?,?,?,?,?,?,?)`, r.ID, r.JobID, r.ParentRevisionID, r.Sequence, r.AuthorID, r.ChangeNote, string(data), r.ContentDigest, r.SubmittedAt.Format(time.RFC3339Nano))
	return err
}

func scanRevision(row scanner) (domain.CaptionRevision, error) {
	var r domain.CaptionRevision
	var data, at string
	err := row.Scan(&r.ID, &r.JobID, &r.ParentRevisionID, &r.Sequence, &r.AuthorID, &r.ChangeNote, &data, &r.ContentDigest, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return r, domain.NewError(domain.CodeNotFound, "修订不存在")
	}
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal([]byte(data), &r.Segments); err != nil {
		return r, err
	}
	r.SubmittedAt, err = time.Parse(time.RFC3339Nano, at)
	return r, err
}
func (t *transaction) GetRevision(id string) (domain.CaptionRevision, error) {
	return scanRevision(t.tx.QueryRow(`SELECT id,job_id,parent_revision_id,sequence,author_id,change_note,segments_json,content_digest,submitted_at FROM caption_revisions WHERE id=?`, id))
}
func (t *transaction) ListRevisions(jobID string) ([]domain.CaptionRevision, error) {
	rows, err := t.tx.Query(`SELECT id,job_id,parent_revision_id,sequence,author_id,change_note,segments_json,content_digest,submitted_at FROM caption_revisions WHERE job_id=? ORDER BY sequence`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.CaptionRevision{}
	for rows.Next() {
		r, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (t *transaction) InsertFindings(items []domain.QualityFinding) error {
	for _, f := range items {
		ids, err := json.Marshal(f.SegmentIDs)
		if err != nil {
			return err
		}
		_, err = t.tx.Exec(`INSERT INTO quality_findings(id,job_id,revision_id,rule_code,severity,segment_ids_json,message,disposition,rationale,resolved_by,resolved_at) VALUES(?,?,?,?,?,?,?,?,?,?,NULL)`, f.ID, f.JobID, f.RevisionID, f.RuleCode, f.Severity, string(ids), f.Message, f.Disposition, f.Rationale, f.ResolvedBy)
		if err != nil {
			return err
		}
	}
	return nil
}
func scanFinding(row scanner) (domain.QualityFinding, error) {
	var f domain.QualityFinding
	var ids string
	var at sql.NullString
	err := row.Scan(&f.ID, &f.JobID, &f.RevisionID, &f.RuleCode, &f.Severity, &ids, &f.Message, &f.Disposition, &f.Rationale, &f.ResolvedBy, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return f, domain.NewError(domain.CodeNotFound, "问题不存在")
	}
	if err != nil {
		return f, err
	}
	if err = json.Unmarshal([]byte(ids), &f.SegmentIDs); err != nil {
		return f, err
	}
	if at.Valid {
		x, e := time.Parse(time.RFC3339Nano, at.String)
		if e != nil {
			return f, e
		}
		f.ResolvedAt = &x
	}
	return f, nil
}

const findingColumns = `id,job_id,revision_id,rule_code,severity,segment_ids_json,message,disposition,rationale,resolved_by,resolved_at`

func (t *transaction) GetFinding(id string) (domain.QualityFinding, error) {
	return scanFinding(t.tx.QueryRow(`SELECT `+findingColumns+` FROM quality_findings WHERE id=?`, id))
}
func (t *transaction) ListFindings(jobID, revisionID string) ([]domain.QualityFinding, error) {
	query := `SELECT ` + findingColumns + ` FROM quality_findings WHERE job_id=?`
	args := []any{jobID}
	if revisionID != "" {
		query += ` AND revision_id=?`
		args = append(args, revisionID)
	}
	query += ` ORDER BY rowid`
	rows, err := t.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.QualityFinding{}
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}
func (t *transaction) UpdateFinding(f domain.QualityFinding) error {
	var at any
	if f.ResolvedAt != nil {
		at = f.ResolvedAt.Format(time.RFC3339Nano)
	}
	result, err := t.tx.Exec(`UPDATE quality_findings SET disposition=?,rationale=?,resolved_by=?,resolved_at=? WHERE id=? AND disposition=?`, f.Disposition, f.Rationale, f.ResolvedBy, at, f.ID, domain.DispositionOpen)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return domain.NewError(domain.CodeConflict, "问题已由其他操作处置")
	}
	return nil
}

func (t *transaction) UpdateFindings(items []domain.QualityFinding) error {
	for _, f := range items {
		if err := t.UpdateFinding(f); err != nil {
			return err
		}
	}
	return nil
}

func (t *transaction) InsertReview(r domain.ReviewDecision) error {
	ids, err := json.Marshal(r.SampledSegmentIDs)
	if err != nil {
		return err
	}
	_, err = t.tx.Exec(`INSERT INTO review_decisions(id,job_id,revision_id,reviewer_id,sampled_segment_ids_json,decision,comment,decided_at) VALUES(?,?,?,?,?,?,?,?)`, r.ID, r.JobID, r.RevisionID, r.ReviewerID, string(ids), r.Decision, r.Comment, r.DecidedAt.Format(time.RFC3339Nano))
	return err
}
func (t *transaction) ListReviews(jobID string) ([]domain.ReviewDecision, error) {
	rows, err := t.tx.Query(`SELECT id,job_id,revision_id,reviewer_id,sampled_segment_ids_json,decision,comment,decided_at FROM review_decisions WHERE job_id=? ORDER BY decided_at`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.ReviewDecision{}
	for rows.Next() {
		var r domain.ReviewDecision
		var ids, at string
		if err := rows.Scan(&r.ID, &r.JobID, &r.RevisionID, &r.ReviewerID, &ids, &r.Decision, &r.Comment, &at); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(ids), &r.SampledSegmentIDs); err != nil {
			return nil, err
		}
		r.DecidedAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
