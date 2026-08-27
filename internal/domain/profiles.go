package domain

import "strings"

type RuleSnapshot struct {
	ProfileCode       string  `json:"profileCode"`
	Version           string  `json:"version"`
	MaxCharsPerSecond float64 `json:"maxCharsPerSecond"`
	MaxLineRunes      int     `json:"maxLineRunes"`
	MinDurationMillis int64   `json:"minDurationMillis"`
	MaxDurationMillis int64   `json:"maxDurationMillis"`
	MaxGapMillis      int64   `json:"maxGapMillis"`
}

func RuleSetForProfile(profile string) (QualityRuleSet, RuleSnapshot, error) {
	p := strings.ToUpper(strings.TrimSpace(profile))
	r := DefaultRuleSet()
	switch p {
	case "PUBLIC-CULTURE-V1", "P1", "P":
	case "BROADCAST-V1":
		r.MaxCharsPerSecond = 17
		r.MaxLineRunes = 42
		r.MinDurationMillis = 500
		r.MaxDurationMillis = 8000
		r.MaxGapMillis = 5000
	default:
		return QualityRuleSet{}, RuleSnapshot{}, NewError(CodeValidation, "未知或停用的发布规范: %s", profile)
	}
	return r, RuleSnapshot{ProfileCode: p, Version: RuleVersion, MaxCharsPerSecond: r.MaxCharsPerSecond, MaxLineRunes: r.MaxLineRunes, MinDurationMillis: r.MinDurationMillis, MaxDurationMillis: r.MaxDurationMillis, MaxGapMillis: r.MaxGapMillis}, nil
}
