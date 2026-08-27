package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

func ValidateNewJob(job CaptionJob) error {
	if strings.TrimSpace(job.Title) == "" {
		return NewError(CodeValidation, "片名不能为空")
	}
	if strings.TrimSpace(job.MediaRef) == "" {
		return NewError(CodeValidation, "媒体引用不能为空")
	}
	if job.DurationMillis <= 0 {
		return NewError(CodeValidation, "节目时长必须大于零")
	}
	if strings.TrimSpace(job.SourceLanguage) == "" || strings.TrimSpace(job.TargetLanguage) == "" {
		return NewError(CodeValidation, "源语言和目标语言不能为空")
	}
	if strings.TrimSpace(job.ProfileCode) == "" {
		return NewError(CodeValidation, "发布规范不能为空")
	}
	return nil
}

func ValidateSegments(segments []Segment, duration int64) error {
	if len(segments) == 0 {
		return NewError(CodeValidation, "修订必须至少包含一个字幕片段")
	}
	seen := map[string]bool{}
	var previousStart int64 = -1
	for i, segment := range segments {
		if strings.TrimSpace(segment.ID) == "" {
			return NewError(CodeValidation, "第 %d 个片段缺少 ID", i+1)
		}
		if seen[segment.ID] {
			return NewError(CodeValidation, "片段 ID %s 重复", segment.ID)
		}
		seen[segment.ID] = true
		if segment.StartMillis < 0 || segment.EndMillis <= segment.StartMillis {
			return NewError(CodeValidation, "片段 %s 的时间边界无效", segment.ID)
		}
		if segment.EndMillis > duration {
			return NewError(CodeValidation, "片段 %s 超出媒体边界", segment.ID)
		}
		if segment.StartMillis < previousStart {
			return NewError(CodeValidation, "片段必须按开始时间排序")
		}
		previousStart = segment.StartMillis
	}
	return nil
}

func RevisionDigest(segments []Segment) string {
	data, _ := json.Marshal(segments)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func SegmentSet(segments []Segment) map[string]Segment {
	set := make(map[string]Segment, len(segments))
	for _, segment := range segments {
		set[segment.ID] = segment
	}
	return set
}
