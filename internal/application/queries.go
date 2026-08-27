package application

import (
	"context"
	"sort"
	"strings"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/evidence"
)

func (s *Service) ListJobs(ctx context.Context, limit, offset int) ([]domain.CaptionJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var result []domain.CaptionJob
	err := s.repo.Transact(ctx, func(tx Tx) error { var err error; result, err = tx.ListJobs(limit, offset); return err })
	return result, err
}

func (s *Service) JobDetail(ctx context.Context, id string) (JobDetail, error) {
	var result JobDetail
	err := s.repo.Transact(ctx, func(tx Tx) error {
		job, err := tx.GetJob(id)
		if err != nil {
			return err
		}
		result.Job = job
		result.Revisions, err = tx.ListRevisions(id)
		if err != nil {
			return err
		}
		result.Findings, err = tx.ListFindings(id, "")
		if err != nil {
			return err
		}
		result.Reviews, err = tx.ListReviews(id)
		if err != nil {
			return err
		}
		artifact, err := tx.GetArtifactByJob(id)
		if err == nil {
			result.Artifact = &artifact
		} else if domain.ErrorCodeOf(err) != domain.CodeNotFound {
			return err
		}
		return nil
	})
	return result, err
}

func (s *Service) Timeline(ctx context.Context, jobID string, limit, offset int) ([]domain.AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var events []domain.AuditEvent
	err := s.repo.Transact(ctx, func(tx Tx) error { var err error; events, err = tx.ListAudit(jobID, limit, offset); return err })
	return events, err
}

func (s *Service) Diff(ctx context.Context, jobID, fromID, toID string) (RevisionDiff, error) {
	var result RevisionDiff
	err := s.repo.Transact(ctx, func(tx Tx) error {
		from, err := tx.GetRevision(fromID)
		if err != nil {
			return err
		}
		to, err := tx.GetRevision(toID)
		if err != nil {
			return err
		}
		if from.JobID != jobID || to.JobID != jobID {
			return domain.NewError(domain.CodeNotFound, "修订不属于该任务")
		}
		if to.ParentRevisionID != from.ID {
			return domain.NewError(domain.CodeConflict, "只能对比直接相邻的父子修订")
		}
		result = RevisionDiff{FromRevisionID: fromID, ToRevisionID: toID, Changes: []SegmentChange{}}
		before, after := domain.SegmentSet(from.Segments), domain.SegmentSet(to.Segments)
		ids := map[string]bool{}
		for id := range before {
			ids[id] = true
		}
		for id := range after {
			ids[id] = true
		}
		ordered := make([]string, 0, len(ids))
		for id := range ids {
			ordered = append(ordered, id)
		}
		sort.Strings(ordered)
		for _, id := range ordered {
			b, bok := before[id]
			a, aok := after[id]
			change := SegmentChange{SegmentID: id}
			if !bok {
				change.Kind = "added"
				change.After = &a
			} else if !aok {
				change.Kind = "removed"
				change.Before = &b
			} else if b != a {
				change.Kind = "changed"
				change.Before = &b
				change.After = &a
			} else {
				continue
			}
			result.Changes = append(result.Changes, change)
		}
		ff, err := tx.ListFindings(jobID, fromID)
		if err != nil {
			return err
		}
		tf, err := tx.ListFindings(jobID, toID)
		if err != nil {
			return err
		}
		result.FromFindingCount, result.ToFindingCount = len(ff), len(tf)
		result.FindingChanges = []FindingChange{}
		beforeF := map[string]domain.QualityFinding{}
		afterF := map[string]domain.QualityFinding{}
		for _, f := range ff {
			for _, sid := range f.SegmentIDs {
				beforeF[f.RuleCode+"|"+sid] = f
			}
		}
		for _, f := range tf {
			for _, sid := range f.SegmentIDs {
				afterF[f.RuleCode+"|"+sid] = f
			}
		}
		keys := map[string]bool{}
		for k := range beforeF {
			keys[k] = true
		}
		for k := range afterF {
			keys[k] = true
		}
		for k := range keys {
			_, b := beforeF[k]
			_, a := afterF[k]
			kind := ""
			if b && !a {
				kind = "resolved"
			} else if !b && a {
				kind = "new"
			} else if b && a {
				kind = "still_present"
			}
			if kind != "" {
				f := afterF[k]
				if !a {
					f = beforeF[k]
				}
				parts := strings.SplitN(k, "|", 2)
				result.FindingChanges = append(result.FindingChanges, FindingChange{RuleCode: parts[0], SegmentID: parts[1], Kind: kind, Severity: f.Severity})
			}
		}
		sort.Slice(result.FindingChanges, func(i, j int) bool {
			return result.FindingChanges[i].RuleCode+result.FindingChanges[i].SegmentID < result.FindingChanges[j].RuleCode+result.FindingChanges[j].SegmentID
		})
		return nil
	})
	return result, err
}

func (s *Service) Verify(ctx context.Context, jobID string) (evidence.VerificationResult, error) {
	var result evidence.VerificationResult
	err := s.repo.Transact(ctx, func(tx Tx) error {
		artifact, err := tx.GetArtifactByJob(jobID)
		if err != nil {
			return err
		}
		revision, err := tx.GetRevision(artifact.RevisionID)
		if err != nil {
			return err
		}
		events, err := tx.ListAudit(jobID, 10000, 0)
		if err != nil {
			return err
		}
		result = evidence.VerifyRelease(revision, artifact, events)
		return nil
	})
	return result, err
}

func (s *Service) Artifact(ctx context.Context, jobID string) (domain.ReleaseArtifact, error) {
	var result domain.ReleaseArtifact
	err := s.repo.Transact(ctx, func(tx Tx) error { var err error; result, err = tx.GetArtifactByJob(jobID); return err })
	return result, err
}
