package sqlite

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

const jobColumns = `id,title,media_ref,duration_millis,source_language,target_language,profile_code,status,current_revision_id,version,created_at,updated_at`

type scanner interface{ Scan(...any) error }

func scanJob(row scanner) (domain.CaptionJob, error) {
	var job domain.CaptionJob
	var created, updated string
	err := row.Scan(&job.ID, &job.Title, &job.MediaRef, &job.DurationMillis, &job.SourceLanguage, &job.TargetLanguage, &job.ProfileCode, &job.Status, &job.CurrentRevisionID, &job.Version, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return job, domain.NewError(domain.CodeNotFound, "任务不存在")
	}
	if err != nil {
		return job, err
	}
	job.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return job, err
	}
	job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return job, err
}

func (t *transaction) ListJobsFiltered(f application.JobQueueFilter) ([]domain.CaptionJob, int, map[domain.JobStatus]int, error) {
	query := ` FROM caption_jobs WHERE 1=1`
	args := []any{}
	if f.Keyword != "" {
		query += ` AND (LOWER(title) LIKE LOWER(?) OR LOWER(media_ref) LIKE LOWER(?))`
		k := "%" + strings.ToLower(f.Keyword) + "%"
		args = append(args, k, k)
	}
	if f.Status != "" {
		query += ` AND status=?`
		args = append(args, f.Status)
	}
	if f.ProfileCode != "" {
		query += ` AND profile_code=?`
		args = append(args, f.ProfileCode)
	}
	if f.SourceLanguage != "" {
		query += ` AND source_language=?`
		args = append(args, f.SourceLanguage)
	}
	if f.TargetLanguage != "" {
		query += ` AND target_language=?`
		args = append(args, f.TargetLanguage)
	}
	var total int
	if err := t.tx.QueryRow(`SELECT COUNT(*)`+query, args...).Scan(&total); err != nil {
		return nil, 0, nil, err
	}
	counts := map[domain.JobStatus]int{}
	rows, err := t.tx.Query(`SELECT status,COUNT(*)`+query+` GROUP BY status`, args...)
	if err != nil {
		return nil, 0, nil, err
	}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			rows.Close()
			return nil, 0, nil, err
		}
		counts[domain.JobStatus(s)] = n
	}
	rows.Close()
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err = t.tx.Query(`SELECT `+jobColumns+query+` ORDER BY updated_at DESC,id LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, nil, err
	}
	defer rows.Close()
	items := []domain.CaptionJob{}
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			return nil, 0, nil, e
		}
		items = append(items, j)
	}
	return items, total, counts, rows.Err()
}

func (t *transaction) GetJob(id string) (domain.CaptionJob, error) {
	return scanJob(t.tx.QueryRow(`SELECT `+jobColumns+` FROM caption_jobs WHERE id=?`, id))
}

func (t *transaction) ListJobs(limit, offset int) ([]domain.CaptionJob, error) {
	rows, err := t.tx.Query(`SELECT `+jobColumns+` FROM caption_jobs ORDER BY updated_at DESC,id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.CaptionJob{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, job)
	}
	return result, rows.Err()
}

func (t *transaction) InsertJob(job domain.CaptionJob) error {
	_, err := t.tx.Exec(`INSERT INTO caption_jobs (`+jobColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, job.ID, job.Title, job.MediaRef, job.DurationMillis, job.SourceLanguage, job.TargetLanguage, job.ProfileCode, job.Status, job.CurrentRevisionID, job.Version, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (t *transaction) UpdateJob(job domain.CaptionJob, expected int64) error {
	result, err := t.tx.Exec(`UPDATE caption_jobs SET title=?,media_ref=?,duration_millis=?,source_language=?,target_language=?,profile_code=?,status=?,current_revision_id=?,version=?,updated_at=? WHERE id=? AND version=?`, job.Title, job.MediaRef, job.DurationMillis, job.SourceLanguage, job.TargetLanguage, job.ProfileCode, job.Status, job.CurrentRevisionID, job.Version, job.UpdatedAt.Format(time.RFC3339Nano), job.ID, expected)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return domain.NewError(domain.CodeConflict, "任务版本已被其他操作更新")
	}
	return nil
}
