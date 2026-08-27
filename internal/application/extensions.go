package application

import (
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/evidence"
	"context"
	"encoding/json"
	"sort"
	"strings"
)

type VerifyCommand struct {
	JobID, VerifierID, IdempotencyKey string
	ExpectedVersion                   int64
}

func (s *Service) VerifyAndRecord(ctx context.Context, cmd VerifyCommand) (domain.VerificationRecord, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return domain.VerificationRecord{}, err
	}
	var out domain.VerificationRecord
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if v, ok, e := idemLoad[domain.VerificationRecord](tx, "verify", cmd.IdempotencyKey); e != nil {
			return e
		} else if ok {
			out = v
			return nil
		}
		if strings.TrimSpace(cmd.VerifierID) == "" {
			return domain.NewError(domain.CodeValidation, "验证人不能为空")
		}
		job, e := tx.GetJob(cmd.JobID)
		if e != nil {
			return e
		}
		if e = expectVersion(job, cmd.ExpectedVersion); e != nil {
			return e
		}
		a, e := tx.GetArtifactByJob(cmd.JobID)
		if e != nil {
			return e
		}
		r, e := tx.GetRevision(a.RevisionID)
		if e != nil {
			return e
		}
		events, e := tx.ListAudit(cmd.JobID, 100000, 0)
		if e != nil {
			return e
		}
		vr := evidence.VerifyRelease(r, a, events)
		now := s.clock()
		out = domain.VerificationRecord{ID: newID("verify"), JobID: cmd.JobID, ArtifactID: a.ID, VerifierID: cmd.VerifierID, Valid: vr.Valid, Reasons: vr.Reasons, WebVTTDigest: vr.WebVTTDigest, ManifestDigest: vr.ManifestHash, AuditHeadDigest: vr.AuditHead, CreatedAt: now}
		if e = tx.InsertVerification(out); e != nil {
			return e
		}
		status := "failed"
		if vr.Valid {
			status = "verified"
		}
		if e = tx.UpdateArtifactStatus(cmd.JobID, status); e != nil {
			return e
		}
		job.Version++
		job.UpdatedAt = now
		if e = tx.UpdateJob(job, cmd.ExpectedVersion); e != nil {
			return e
		}
		if e = appendAudit(tx, cmd.JobID, "release.verified", cmd.VerifierID, out, now); e != nil {
			return e
		}
		return idemSave(tx, "verify", cmd.IdempotencyKey, out)
	})
	return out, err
}
func (s *Service) VerificationHistory(ctx context.Context, jobID string, limit, offset int) ([]domain.VerificationRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		return nil, domain.NewError(domain.CodeValidation, "分页偏移量不能为负数")
	}
	var out []domain.VerificationRecord
	err := s.repo.Transact(ctx, func(tx Tx) error { var e error; out, e = tx.ListVerifications(jobID, limit, offset); return e })
	return out, err
}

func (s *Service) QueryWorkQueue(ctx context.Context, f JobQueueFilter) (JobQueue, error) {
	if len([]rune(f.Keyword)) > 100 {
		return JobQueue{}, domain.NewError(domain.CodeValidation, "关键词长度不能超过100个字符")
	}
	valid := map[domain.JobStatus]bool{domain.StatusDraft: true, domain.StatusRemediating: true, domain.StatusReviewReady: true, domain.StatusApproved: true, domain.StatusFrozen: true}
	if f.Status != "" && !valid[domain.JobStatus(f.Status)] {
		return JobQueue{}, domain.NewError(domain.CodeValidation, "未知任务状态: %s", f.Status)
	}
	if f.Limit < 0 || f.Offset < 0 {
		return JobQueue{}, domain.NewError(domain.CodeValidation, "分页参数不能为负数")
	}
	if f.Limit > 100 {
		return JobQueue{}, domain.NewError(domain.CodeValidation, "分页大小不能超过100")
	}
	var q JobQueue
	err := s.repo.Transact(ctx, func(tx Tx) error { var e error; q.Items, q.Total, q.StatusCounts, e = tx.ListJobsFiltered(f); return e })
	q.Limit = f.Limit
	if q.Limit <= 0 {
		q.Limit = 50
	}
	q.Offset = f.Offset
	return q, err
}

func summarize(a, b []domain.Segment) ChangeSummary {
	bm, am := domain.SegmentSet(a), domain.SegmentSet(b)
	ids := map[string]bool{}
	for id := range bm {
		ids[id] = true
	}
	for id := range am {
		ids[id] = true
	}
	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	c := ChangeSummary{}
	for _, id := range keys {
		x, xok := bm[id]
		y, yok := am[id]
		if !xok {
			c.Added++
			continue
		}
		if !yok {
			c.Removed++
			continue
		}
		if x.Text != y.Text {
			c.TextChanged++
		}
		if x.StartMillis != y.StartMillis || x.EndMillis != y.EndMillis {
			c.TimingChanged++
		}
		if x.Speaker != y.Speaker {
			c.SpeakerChanged++
		}
	}
	data, _ := json.Marshal(c)
	c.Digest = evidence.SHA256(data)
	return c
}

func (s *Service) PreflightRevision(ctx context.Context, cmd SubmitRevisionCommand) (RevisionPreflight, error) {
	s.preflightMu.RLock()
	cached, ok := s.preflightByJob[cmd.JobID]
	s.preflightMu.RUnlock()
	if ok {
		return cached, nil
	}

	var out RevisionPreflight
	err := s.repo.Transact(ctx, func(tx Tx) error {
		job, e := tx.GetJob(cmd.JobID)
		if e != nil {
			return e
		}
		if e = expectVersion(job, cmd.ExpectedVersion); e != nil {
			return e
		}
		if e = domain.EnsureMutable(job); e != nil {
			return e
		}
		revs, e := tx.ListRevisions(job.ID)
		if e != nil {
			return e
		}
		seq := len(revs) + 1
		if e = domain.ValidateNewRevision(job, cmd.ParentRevisionID, seq); e != nil {
			return e
		}
		if e = domain.ValidateSegments(cmd.Segments, job.DurationMillis); e != nil {
			return e
		}
		var parent []domain.Segment
		if cmd.ParentRevisionID != "" {
			p, e := tx.GetRevision(cmd.ParentRevisionID)
			if e != nil {
				return e
			}
			parent = p.Segments
		}
		digest := domain.RevisionDigest(cmd.Segments)
		out = RevisionPreflight{JobID: job.ID, ParentRevisionID: cmd.ParentRevisionID, ContentDigest: digest, ExpectedVersion: job.Version, Sequence: seq, Summary: summarize(parent, cmd.Segments)}
		return nil
	})
	if err == nil {
		s.preflightMu.Lock()
		s.preflightByJob[cmd.JobID] = out
		s.preflightMu.Unlock()
	}
	return out, err
}

func (s *Service) BatchDisposition(ctx context.Context, cmd BatchDispositionCommand) ([]domain.QualityFinding, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return nil, err
	}
	if len(cmd.Items) == 0 || len(cmd.Items) > 100 {
		return nil, domain.NewError(domain.CodeValidation, "批量处置项目数量无效")
	}
	var out []domain.QualityFinding
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if v, ok, e := idemLoad[[]domain.QualityFinding](tx, "batch_disposition", cmd.IdempotencyKey); e != nil {
			return e
		} else if ok {
			out = v
			return nil
		}
		job, e := tx.GetJob(cmd.JobID)
		if e != nil {
			return e
		}
		if e = expectVersion(job, cmd.ExpectedVersion); e != nil {
			return e
		}
		seen := map[string]bool{}
		items := make([]domain.QualityFinding, 0, len(cmd.Items))
		for _, in := range cmd.Items {
			if seen[in.FindingID] {
				return domain.NewError(domain.CodeValidation, "问题编号重复")
			}
			seen[in.FindingID] = true
			f, e := tx.GetFinding(in.FindingID)
			if e != nil {
				return e
			}
			if f.JobID != job.ID || f.RevisionID != job.CurrentRevisionID {
				return domain.NewError(domain.CodeConflict, "问题不属于当前修订")
			}
			if e = domain.ValidateDisposition(f, in.Disposition, in.Rationale, cmd.ActorID); e != nil {
				return e
			}
			f.Disposition = in.Disposition
			f.Rationale = in.Rationale
			f.ResolvedBy = cmd.ActorID
			now := s.clock()
			f.ResolvedAt = &now
			items = append(items, f)
		}
		if e = tx.UpdateFindings(items); e != nil {
			return e
		}
		all, e := tx.ListFindings(job.ID, job.CurrentRevisionID)
		if e != nil {
			return e
		}
		job.Status = domain.StatusReviewReady
		if !domain.FindingsPermitReview(all) {
			job.Status = domain.StatusRemediating
		}
		now := s.clock()
		job.Version++
		job.UpdatedAt = now
		if e = tx.UpdateJob(job, cmd.ExpectedVersion); e != nil {
			return e
		}
		if e = appendAudit(tx, job.ID, "finding.batch_dispositioned", cmd.ActorID, map[string]any{"items": cmd.Items}, now); e != nil {
			return e
		}
		if e = idemSave(tx, "batch_disposition", cmd.IdempotencyKey, items); e != nil {
			return e
		}
		out = items
		return nil
	})
	return out, err
}

func (s *Service) Sampling(ctx context.Context, jobID, revisionID string) (SamplingSuggestion, error) {
	var out SamplingSuggestion
	err := s.repo.Transact(ctx, func(tx Tx) error {
		job, e := tx.GetJob(jobID)
		if e != nil {
			return e
		}
		_ = job
		r, e := tx.GetRevision(revisionID)
		if e != nil {
			return e
		}
		if r.JobID != jobID {
			return domain.NewError(domain.CodeNotFound, "修订不属于该任务")
		}
		n := len(r.Segments)
		min := 3
		if n < min {
			min = n
		}
		ids := []string{}
		if n > 0 {
			ids = append(ids, r.Segments[0].ID)
		}
		if n > 1 {
			ids = append(ids, r.Segments[n-1].ID)
		}
		if n > 2 {
			ids = append(ids, r.Segments[n/2].ID)
		}
		for _, f := range mustFindings(tx, jobID, revisionID) {
			if f.Severity == "blocking" && len(f.SegmentIDs) > 0 {
				ids = append(ids, f.SegmentIDs[0])
			}
		}
		uniq := map[string]bool{}
		u := []string{}
		for _, id := range ids {
			if !uniq[id] {
				uniq[id] = true
				u = append(u, id)
			}
		}
		if len(u) < min {
			for _, seg := range r.Segments {
				if !uniq[seg.ID] {
					uniq[seg.ID] = true
					u = append(u, seg.ID)
				}
				if len(u) >= min {
					break
				}
			}
		}
		out = SamplingSuggestion{RevisionID: revisionID, SegmentIDs: u, Minimum: min}
		for _, seg := range r.Segments {
			out.TotalMillis += seg.EndMillis - seg.StartMillis
			if uniq[seg.ID] {
				out.CoveredMillis += seg.EndMillis - seg.StartMillis
			}
		}
		return nil
	})
	return out, err
}
func mustFindings(tx Tx, j, r string) []domain.QualityFinding {
	v, _ := tx.ListFindings(j, r)
	return v
}

func (s *Service) FreezePreview(ctx context.Context, jobID, revisionID string, expected int64) (FreezePreview, error) {
	var out FreezePreview
	err := s.repo.Transact(ctx, func(tx Tx) error {
		job, e := tx.GetJob(jobID)
		if e != nil {
			return e
		}
		if e = expectVersion(job, expected); e != nil {
			return e
		}
		if e = domain.ValidateFreeze(job, "preview"); e != nil {
			return e
		}
		r, e := tx.GetRevision(revisionID)
		if e != nil {
			return e
		}
		if r.JobID != jobID || job.CurrentRevisionID != revisionID {
			return domain.NewError(domain.CodeConflict, "只能预览当前修订")
		}
		reviews, e := tx.ListReviews(jobID)
		if e != nil {
			return e
		}
		reviewer := ""
		if len(reviews) > 0 {
			reviewer = reviews[len(reviews)-1].ReviewerID
		}
		head, e := tx.LastAuditDigest(jobID)
		if e != nil {
			return e
		}
		findings, e := tx.ListFindings(jobID, revisionID)
		if e != nil {
			return e
		}
		raw := r.ContentDigest + "|" + head + "|" + reviewer
		out = FreezePreview{JobID: jobID, RevisionID: revisionID, ExpectedVersion: job.Version, RevisionDigest: r.ContentDigest, AuditHeadDigest: head, SummaryDigest: evidence.SHA256([]byte(raw)), Reviewer: reviewer, RuleVersion: domain.RuleVersion}
		for _, f := range findings {
			if f.Disposition == domain.DispositionOpen || f.Disposition == domain.DispositionFix {
				out.Unresolved++
			}
		}
		return nil
	})
	return out, err
}
