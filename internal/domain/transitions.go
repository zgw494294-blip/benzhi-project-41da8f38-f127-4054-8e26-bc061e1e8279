package domain

import "strings"

func EnsureMutable(job CaptionJob) error {
	if job.Status == StatusFrozen {
		return NewError(CodeState, "任务已冻结，禁止修改既有证据")
	}
	return nil
}

func ValidateNewRevision(job CaptionJob, parent string, sequence int) error {
	if err := EnsureMutable(job); err != nil {
		return err
	}
	if job.CurrentRevisionID == "" {
		if parent != "" || sequence != 1 {
			return NewError(CodeConflict, "首个修订必须无父修订且序号为 1")
		}
		return nil
	}
	if parent != job.CurrentRevisionID {
		return NewError(CodeConflict, "继任修订必须指向当前修订")
	}
	if sequence < 2 {
		return NewError(CodeConflict, "继任修订序号无效")
	}
	return nil
}

func ValidateDisposition(finding QualityFinding, disposition Disposition, rationale, actor string) error {
	if finding.Disposition != DispositionOpen {
		return NewError(CodeConflict, "问题已处置，不可覆盖")
	}
	if disposition != DispositionConfirmed && disposition != DispositionFalse && disposition != DispositionFix {
		return NewError(CodeValidation, "问题处置类型无效")
	}
	if strings.TrimSpace(rationale) == "" || strings.TrimSpace(actor) == "" {
		return NewError(CodeValidation, "处置依据和处置人不能为空")
	}
	return nil
}

func ValidateReview(job CaptionJob, revision CaptionRevision, findings []QualityFinding, reviewer, decision, comment string, sampled []string) error {
	if job.Status != StatusReviewReady {
		return NewError(CodeState, "任务尚未达到复核条件")
	}
	if reviewer == revision.AuthorID {
		return NewError(CodeForbidden, "复核员不得是本次修订提交者")
	}
	if !FindingsPermitReview(findings) {
		return NewError(CodeState, "质量问题尚未全部处置，或仍存在未清零的阻断问题")
	}
	if decision != "approve" && decision != "reject" {
		return NewError(CodeValidation, "复核决定必须为 approve 或 reject")
	}
	if strings.TrimSpace(reviewer) == "" || strings.TrimSpace(comment) == "" || len(sampled) == 0 {
		return NewError(CodeValidation, "复核员、抽检片段和意见不能为空")
	}
	known := SegmentSet(revision.Segments)
	seen := map[string]bool{}
	for _, id := range sampled {
		if seen[id] {
			return NewError(CodeValidation, "抽检片段不能重复")
		}
		seen[id] = true
		if _, ok := known[id]; !ok {
			return NewError(CodeValidation, "抽检片段 %s 不存在", id)
		}
	}
	if decision == "approve" {
		minimum := 3
		if len(revision.Segments) < minimum {
			minimum = len(revision.Segments)
		}
		if len(sampled) < minimum {
			return NewError(CodeValidation, "通过复核至少需要抽检 %d 个片段", minimum)
		}
		if len(revision.Segments) > 1 {
			first, last := revision.Segments[0].ID, revision.Segments[len(revision.Segments)-1].ID
			hasFirst, hasLast := false, false
			for _, id := range sampled {
				if id == first {
					hasFirst = true
				}
				if id == last {
					hasLast = true
				}
			}
			if !hasFirst || !hasLast {
				return NewError(CodeValidation, "通过复核必须覆盖首尾时间区间")
			}
		}
	}
	return nil
}

func ValidateFreeze(job CaptionJob, actor string) error {
	if job.Status != StatusApproved {
		return NewError(CodeState, "仅已通过复核的修订可以冻结")
	}
	if strings.TrimSpace(actor) == "" {
		return NewError(CodeValidation, "发布负责人不能为空")
	}
	return nil
}
