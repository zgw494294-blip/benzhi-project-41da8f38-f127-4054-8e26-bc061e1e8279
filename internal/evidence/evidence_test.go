package evidence

import (
	"strings"
	"testing"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

func TestWebVTTIsCanonicalAndStable(t *testing.T) {
	revision := domain.CaptionRevision{Segments: []domain.Segment{{ID: "b", StartMillis: 2000, EndMillis: 3000, Text: "第二条"}, {ID: "a", StartMillis: 0, EndMillis: 1000, Speaker: "旁白", Text: "第一条\r\n续行"}}}
	first, digest1 := BuildWebVTT(revision)
	second, digest2 := BuildWebVTT(revision)
	if string(first) != string(second) || digest1 != digest2 {
		t.Fatal("重复生成结果不一致")
	}
	if !strings.HasPrefix(string(first), "WEBVTT\n\na\n00:00:00.000 --> 00:00:01.000") {
		t.Fatalf("时间轴未稳定排序或格式错误:\n%s", first)
	}
	if strings.Contains(string(first), "\r") {
		t.Fatal("WebVTT 应只使用 LF")
	}
}

func TestAuditChainDetectsTampering(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	one := domain.AuditEvent{JobID: "j", EventType: "created", ActorID: "a", Payload: "{}", OccurredAt: at}
	one.Digest = AuditDigest("", one.JobID, one.EventType, one.ActorID, one.Payload, at)
	two := domain.AuditEvent{JobID: "j", EventType: "changed", ActorID: "b", Payload: "{}", PrevDigest: one.Digest, OccurredAt: at.Add(time.Second)}
	two.Digest = AuditDigest(two.PrevDigest, two.JobID, two.EventType, two.ActorID, two.Payload, two.OccurredAt)
	if head, err := VerifyAudit([]domain.AuditEvent{one, two}); err != nil || head != two.Digest {
		t.Fatalf("有效链校验失败: %v", err)
	}
	two.Payload = `{"tampered":true}`
	if _, err := VerifyAudit([]domain.AuditEvent{one, two}); err == nil {
		t.Fatal("篡改后应校验失败")
	}
}
