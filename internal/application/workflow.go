package application

import (
	"context"
	"encoding/json"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/evidence"
)

func (s *Service) SubmitRevision(ctx context.Context, cmd SubmitRevisionCommand) (domain.CaptionRevision, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return domain.CaptionRevision{}, err
	}
	var result domain.CaptionRevision
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if saved, ok, err := idemLoad[domain.CaptionRevision](tx, "submit_revision", cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			result = saved
			return nil
		}
		job, err := tx.GetJob(cmd.JobID)
		if err != nil {
			return err
		}
		if err := expectVersion(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := domain.EnsureMutable(job); err != nil {
			return err
		}
		revisions, err := tx.ListRevisions(job.ID)
		if err != nil {
			return err
		}
		sequence := len(revisions) + 1
		if err := domain.ValidateNewRevision(job, cmd.ParentRevisionID, sequence); err != nil {
			return err
		}
		if err := domain.ValidateSegments(cmd.Segments, job.DurationMillis); err != nil {
			return err
		}
		if cmd.PreflightDigest != "" {
			if cmd.PreflightVersion != 0 && cmd.PreflightVersion != job.Version {
				return domain.NewError(domain.CodeConflict, "预检所依据的任务版本已变化，请重新预检")
			}
			parentSegments := []domain.Segment{}
			if cmd.ParentRevisionID != "" {
				p, e := tx.GetRevision(cmd.ParentRevisionID)
				if e != nil {
					return e
				}
				parentSegments = p.Segments
			}
			if summarize(parentSegments, cmd.Segments).Digest != cmd.PreflightDigest {
				return domain.NewError(domain.CodeConflict, "提交内容与预检摘要不一致，请重新预检")
			}
		}
		if cmd.AuthorID == "" || cmd.ChangeNote == "" {
			return domain.NewError(domain.CodeValidation, "修订作者和变更说明不能为空")
		}
		now := s.clock()
		revision := domain.CaptionRevision{ID: newID("rev"), JobID: job.ID, ParentRevisionID: cmd.ParentRevisionID, Sequence: sequence, AuthorID: cmd.AuthorID, ChangeNote: cmd.ChangeNote, Segments: append([]domain.Segment(nil), cmd.Segments...), ContentDigest: domain.RevisionDigest(cmd.Segments), SubmittedAt: now}
		if err := tx.InsertRevision(revision); err != nil {
			return err
		}
		job.CurrentRevisionID = revision.ID
		job.Status = domain.StatusDraft
		if sequence > 1 {
			job.Status = domain.StatusRemediating
		}
		job.Version++
		job.UpdatedAt = now
		if err := tx.UpdateJob(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := appendAudit(tx, job.ID, "revision.submitted", cmd.AuthorID, map[string]any{"revisionID": revision.ID, "sequence": sequence, "digest": revision.ContentDigest}, now); err != nil {
			return err
		}
		if err := idemSave(tx, "submit_revision", cmd.IdempotencyKey, revision); err != nil {
			return err
		}
		result = revision
		return nil
	})
	return result, err
}

func (s *Service) RunQuality(ctx context.Context, cmd RunQualityCommand) ([]domain.QualityFinding, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return nil, err
	}
	var result []domain.QualityFinding
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if saved, ok, err := idemLoad[[]domain.QualityFinding](tx, "run_quality", cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			result = saved
			return nil
		}
		job, err := tx.GetJob(cmd.JobID)
		if err != nil {
			return err
		}
		if err := expectVersion(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := domain.EnsureMutable(job); err != nil {
			return err
		}
		if job.CurrentRevisionID != cmd.RevisionID {
			return domain.NewError(domain.CodeConflict, "只能检查当前修订")
		}
		existing, err := tx.ListFindings(job.ID, cmd.RevisionID)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return domain.NewError(domain.CodeConflict, "当前修订已执行质量检查")
		}
		revision, err := tx.GetRevision(cmd.RevisionID)
		if err != nil {
			return err
		}
		rules, snapshot, err := domain.RuleSetForProfile(job.ProfileCode)
		if err != nil {
			return err
		}
		findings := rules.Evaluate(job.ID, revision.ID, revision.Segments)
		for i := range findings {
			findings[i].ID = newID("finding")
		}
		if err := tx.InsertFindings(findings); err != nil {
			return err
		}
		now := s.clock()
		snapshotJSON, _ := json.Marshal(snapshot)
		if err := tx.RecordQualitySnapshot(newID("qc"), job.ID, revision.ID, snapshot.ProfileCode, snapshot.Version, snapshotJSON, now.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		job.Status = domain.StatusReviewReady
		if !domain.FindingsPermitReview(findings) {
			job.Status = domain.StatusRemediating
		}
		job.Version++
		job.UpdatedAt = now
		if err := tx.UpdateJob(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := appendAudit(tx, job.ID, "quality.completed", cmd.ActorID, map[string]any{"revisionID": revision.ID, "findingCount": len(findings), "ruleVersion": snapshot.Version, "ruleSnapshot": snapshot}, now); err != nil {
			return err
		}
		if err := idemSave(tx, "run_quality", cmd.IdempotencyKey, findings); err != nil {
			return err
		}
		result = findings
		return nil
	})
	return result, err
}

func (s *Service) DispositionFinding(ctx context.Context, cmd DispositionCommand) (domain.QualityFinding, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return domain.QualityFinding{}, err
	}
	var result domain.QualityFinding
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if saved, ok, err := idemLoad[domain.QualityFinding](tx, "disposition_finding", cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			result = saved
			return nil
		}
		job, err := tx.GetJob(cmd.JobID)
		if err != nil {
			return err
		}
		if err := expectVersion(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := domain.EnsureMutable(job); err != nil {
			return err
		}
		finding, err := tx.GetFinding(cmd.FindingID)
		if err != nil {
			return err
		}
		if finding.JobID != job.ID || finding.RevisionID != job.CurrentRevisionID {
			return domain.NewError(domain.CodeConflict, "问题不属于当前修订")
		}
		if err := domain.ValidateDisposition(finding, cmd.Disposition, cmd.Rationale, cmd.ActorID); err != nil {
			return err
		}
		now := s.clock()
		finding.Disposition = cmd.Disposition
		finding.Rationale = cmd.Rationale
		finding.ResolvedBy = cmd.ActorID
		finding.ResolvedAt = &now
		if err := tx.UpdateFinding(finding); err != nil {
			return err
		}
		all, err := tx.ListFindings(job.ID, job.CurrentRevisionID)
		if err != nil {
			return err
		}
		job.Status = domain.StatusReviewReady
		if !domain.FindingsPermitReview(all) {
			job.Status = domain.StatusRemediating
		}
		job.Version++
		job.UpdatedAt = now
		if err := tx.UpdateJob(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := appendAudit(tx, job.ID, "finding.dispositioned", cmd.ActorID, map[string]any{"findingID": finding.ID, "disposition": finding.Disposition, "rationale": finding.Rationale}, now); err != nil {
			return err
		}
		if err := idemSave(tx, "disposition_finding", cmd.IdempotencyKey, finding); err != nil {
			return err
		}
		result = finding
		return nil
	})
	return result, err
}

func (s *Service) Review(ctx context.Context, cmd ReviewCommand) (domain.ReviewDecision, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return domain.ReviewDecision{}, err
	}
	var result domain.ReviewDecision
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if saved, ok, err := idemLoad[domain.ReviewDecision](tx, "review", cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			result = saved
			return nil
		}
		job, err := tx.GetJob(cmd.JobID)
		if err != nil {
			return err
		}
		if err := expectVersion(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if job.CurrentRevisionID != cmd.RevisionID {
			return domain.NewError(domain.CodeConflict, "只能复核当前修订")
		}
		revision, err := tx.GetRevision(cmd.RevisionID)
		if err != nil {
			return err
		}
		findings, err := tx.ListFindings(job.ID, revision.ID)
		if err != nil {
			return err
		}
		if err := domain.ValidateReview(job, revision, findings, cmd.ReviewerID, cmd.Decision, cmd.Comment, cmd.SampledSegmentIDs); err != nil {
			return err
		}
		now := s.clock()
		review := domain.ReviewDecision{ID: newID("review"), JobID: job.ID, RevisionID: revision.ID, ReviewerID: cmd.ReviewerID, SampledSegmentIDs: append([]string(nil), cmd.SampledSegmentIDs...), Decision: cmd.Decision, Comment: cmd.Comment, DecidedAt: now}
		if err := tx.InsertReview(review); err != nil {
			return err
		}
		if cmd.Decision == "approve" {
			job.Status = domain.StatusApproved
		} else {
			job.Status = domain.StatusRemediating
		}
		job.Version++
		job.UpdatedAt = now
		if err := tx.UpdateJob(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := appendAudit(tx, job.ID, "review.decided", cmd.ReviewerID, review, now); err != nil {
			return err
		}
		if err := idemSave(tx, "review", cmd.IdempotencyKey, review); err != nil {
			return err
		}
		result = review
		return nil
	})
	return result, err
}

func (s *Service) Freeze(ctx context.Context, cmd FreezeCommand) (domain.ReleaseArtifact, error) {
	if err := validateKey(cmd.IdempotencyKey); err != nil {
		return domain.ReleaseArtifact{}, err
	}
	var result domain.ReleaseArtifact
	err := s.repo.Transact(ctx, func(tx Tx) error {
		if saved, ok, err := idemLoad[domain.ReleaseArtifact](tx, "freeze", cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			result = saved
			return nil
		}
		job, err := tx.GetJob(cmd.JobID)
		if err != nil {
			return err
		}
		if err := expectVersion(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := domain.ValidateFreeze(job, cmd.ActorID); err != nil {
			return err
		}
		if job.CurrentRevisionID != cmd.RevisionID {
			return domain.NewError(domain.CodeConflict, "只能冻结当前修订")
		}
		revision, err := tx.GetRevision(cmd.RevisionID)
		if err != nil {
			return err
		}
		head, err := tx.LastAuditDigest(job.ID)
		if err != nil {
			return err
		}
		if cmd.ConfirmationDigest != "" {
			findings, e := tx.ListFindings(job.ID, revision.ID)
			if e != nil {
				return e
			}
			reviews, e := tx.ListReviews(job.ID)
			if e != nil {
				return e
			}
			reviewer := ""
			if len(reviews) > 0 {
				reviewer = reviews[len(reviews)-1].ReviewerID
			}
			check := evidence.SHA256([]byte(revision.ContentDigest + "|" + head + "|" + reviewer))
			if check != cmd.ConfirmationDigest {
				return domain.NewError(domain.CodeConflict, "放行清单已过期，请重新确认")
			}
			for _, f := range findings {
				if f.Disposition == domain.DispositionOpen || f.Disposition == domain.DispositionFix {
					return domain.NewError(domain.CodeState, "仍有未决质量问题")
				}
			}
		}
		now := s.clock()
		vtt, vttDigest := evidence.BuildWebVTT(revision)
		credential := evidence.CredentialNumber(job.ID, revision.ID, vttDigest)
		manifest, manifestDigest, err := evidence.BuildManifest(job, revision, cmd.ActorID, head, credential, now)
		if err != nil {
			return err
		}
		artifact := domain.ReleaseArtifact{ID: newID("artifact"), JobID: job.ID, RevisionID: revision.ID, CredentialNumber: credential, WebVTT: string(vtt), Manifest: string(manifest), WebVTTDigest: vttDigest, ManifestDigest: manifestDigest, AuditHeadDigest: head, FrozenBy: cmd.ActorID, FrozenAt: now, VerificationStatus: "pending"}
		if err := tx.InsertArtifact(artifact); err != nil {
			return err
		}
		job.Status = domain.StatusFrozen
		job.Version++
		job.UpdatedAt = now
		if err := tx.UpdateJob(job, cmd.ExpectedVersion); err != nil {
			return err
		}
		if err := appendAudit(tx, job.ID, "release.frozen", cmd.ActorID, map[string]any{"artifactID": artifact.ID, "credentialNumber": credential, "manifestDigest": manifestDigest}, now); err != nil {
			return err
		}
		if err := idemSave(tx, "freeze", cmd.IdempotencyKey, artifact); err != nil {
			return err
		}
		result = artifact
		return nil
	})
	return result, err
}

var _ = time.RFC3339
var _ = evidence.VerificationResult{}
