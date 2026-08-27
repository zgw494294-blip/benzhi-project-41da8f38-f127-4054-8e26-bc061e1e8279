package domain

import "testing"

func TestQualityRulesAreDeterministicAndLocateSegments(t *testing.T) {
	segments := []Segment{
		{ID: "a", StartMillis: 0, EndMillis: 500, Text: ""},
		{ID: "b", StartMillis: 400, EndMillis: 900, Text: "这是一段阅读速度一定会超过阈值的很长很长的字幕文本"},
		{ID: "c", StartMillis: 6000, EndMillis: 14000, Text: "没有声音提示"},
	}
	first := DefaultRuleSet().Evaluate("job", "revision", segments)
	second := DefaultRuleSet().Evaluate("job", "revision", segments)
	if len(first) < 7 {
		t.Fatalf("预期覆盖多类规则，实际得到 %d 项: %#v", len(first), first)
	}
	if len(first) != len(second) {
		t.Fatalf("重复检查结果数量不一致")
	}
	for i := range first {
		if first[i].RuleCode != second[i].RuleCode || first[i].Message != second[i].Message {
			t.Fatalf("第 %d 个检查结果不稳定", i)
		}
		if len(first[i].SegmentIDs) == 0 {
			t.Fatalf("问题 %s 缺少片段定位", first[i].RuleCode)
		}
	}
	for _, code := range []string{"EMPTY_TEXT", "OVERLAP", "READING_SPEED", "EXCESSIVE_GAP", "DURATION_SHORT", "DURATION_LONG", "SOUND_CUE"} {
		found := false
		for _, item := range first {
			if item.RuleCode == code {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("缺少规则 %s", code)
		}
	}
}

func TestValidateSegmentsAllowsOverlapForQualityDetection(t *testing.T) {
	segments := []Segment{{ID: "a", StartMillis: 0, EndMillis: 2000, Text: "a"}, {ID: "b", StartMillis: 1000, EndMillis: 3000, Text: "b"}}
	if err := ValidateSegments(segments, 5000); err != nil {
		t.Fatalf("重叠应由质量规则报告，不应阻止修订提交: %v", err)
	}
}

func TestIndependentReviewAndFreezeTransitions(t *testing.T) {
	job := CaptionJob{Status: StatusReviewReady}
	revision := CaptionRevision{AuthorID: "editor", Segments: []Segment{{ID: "s1"}}}
	if err := ValidateReview(job, revision, nil, "editor", "approve", "通过", []string{"s1"}); ErrorCodeOf(err) != CodeForbidden {
		t.Fatalf("应拒绝自审，得到 %v", err)
	}
	if err := ValidateFreeze(job, "publisher"); ErrorCodeOf(err) != CodeState {
		t.Fatalf("未通过复核不应冻结，得到 %v", err)
	}
	job.Status = StatusApproved
	if err := ValidateFreeze(job, "publisher"); err != nil {
		t.Fatalf("已通过修订应允许冻结: %v", err)
	}
}

func TestBlockingConfirmationDoesNotClearFinding(t *testing.T) {
	findings := []QualityFinding{{Severity: "blocking", Disposition: DispositionConfirmed}}
	if FindingsPermitReview(findings) {
		t.Fatal("已确认的阻断问题仍要求继任修订，不应放行复核")
	}
	findings[0].Disposition = DispositionFalse
	if !FindingsPermitReview(findings) {
		t.Fatal("阻断问题被判定为误报后应允许放行")
	}
	findings = []QualityFinding{{Severity: "warning", Disposition: DispositionOpen}}
	if FindingsPermitReview(findings) {
		t.Fatal("提醒问题也必须逐项处置")
	}
}
