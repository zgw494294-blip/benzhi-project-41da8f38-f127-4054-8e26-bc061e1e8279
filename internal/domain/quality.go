package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const RuleVersion = "accessibility-caption-rules-2026.1"

type QualityRuleSet struct {
	MaxCharsPerSecond float64
	MaxLineRunes      int
	MinDurationMillis int64
	MaxDurationMillis int64
	MaxGapMillis      int64
}

func DefaultRuleSet() QualityRuleSet {
	return QualityRuleSet{MaxCharsPerSecond: 15, MaxLineRunes: 32, MinDurationMillis: 700, MaxDurationMillis: 7000, MaxGapMillis: 4000}
}

func (r QualityRuleSet) Evaluate(jobID, revisionID string, segments []Segment) []QualityFinding {
	findings := make([]QualityFinding, 0)
	add := func(rule, severity, message string, ids ...string) {
		findings = append(findings, QualityFinding{JobID: jobID, RevisionID: revisionID, RuleCode: rule, Severity: severity, SegmentIDs: ids, Message: message, Disposition: DispositionOpen})
	}
	for i, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		duration := segment.EndMillis - segment.StartMillis
		if text == "" {
			add("EMPTY_TEXT", "blocking", "字幕文本为空", segment.ID)
		}
		if duration < r.MinDurationMillis {
			add("DURATION_SHORT", "warning", fmt.Sprintf("持续时间 %dms 低于 %dms", duration, r.MinDurationMillis), segment.ID)
		}
		if duration > r.MaxDurationMillis {
			add("DURATION_LONG", "warning", fmt.Sprintf("持续时间 %dms 超过 %dms", duration, r.MaxDurationMillis), segment.ID)
		}
		for _, line := range strings.Split(text, "\n") {
			if utf8.RuneCountInString(line) > r.MaxLineRunes {
				add("LINE_LENGTH", "warning", fmt.Sprintf("单行超过 %d 个字符", r.MaxLineRunes), segment.ID)
				break
			}
		}
		if duration > 0 {
			cps := float64(utf8.RuneCountInString(strings.ReplaceAll(text, "\n", ""))) * 1000 / float64(duration)
			if cps > r.MaxCharsPerSecond {
				add("READING_SPEED", "blocking", fmt.Sprintf("阅读速度 %.1f 字/秒超过 %.1f", cps, r.MaxCharsPerSecond), segment.ID)
			}
		}
		if i > 0 {
			previous := segments[i-1]
			if segment.StartMillis < previous.EndMillis {
				add("OVERLAP", "blocking", "字幕时间码发生重叠", previous.ID, segment.ID)
			} else if segment.StartMillis-previous.EndMillis > r.MaxGapMillis {
				add("EXCESSIVE_GAP", "warning", fmt.Sprintf("字幕间隙超过 %dms", r.MaxGapMillis), previous.ID, segment.ID)
			}
		}
	}
	if !hasSoundCue(segments) {
		ids := []string{}
		if len(segments) > 0 {
			ids = append(ids, segments[0].ID)
		}
		add("SOUND_CUE", "warning", "未检测到必要的声音提示（例如 [音乐]）", ids...)
	}
	return findings
}

func hasSoundCue(segments []Segment) bool {
	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if strings.Contains(text, "[") && strings.Contains(text, "]") || strings.Contains(text, "【") && strings.Contains(text, "】") {
			return true
		}
	}
	return false
}

func HasBlockingOpen(findings []QualityFinding) bool {
	for _, f := range findings {
		if f.Severity == "blocking" && f.Disposition != DispositionFalse {
			return true
		}
	}
	return false
}

func FindingsPermitReview(findings []QualityFinding) bool {
	for _, finding := range findings {
		if finding.Disposition == DispositionOpen || finding.Disposition == DispositionFix {
			return false
		}
		if finding.Severity == "blocking" && finding.Disposition != DispositionFalse {
			return false
		}
	}
	return true
}
