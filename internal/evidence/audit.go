package evidence

import (
	"fmt"
	"strings"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

func AuditDigest(previous, jobID, eventType, actor, payload string, at time.Time) string {
	canonical := strings.Join([]string{previous, jobID, eventType, actor, payload, at.UTC().Format(time.RFC3339Nano)}, "\n")
	return SHA256([]byte(canonical))
}

func VerifyAudit(events []domain.AuditEvent) (string, error) {
	previous := ""
	for i, event := range events {
		if event.PrevDigest != previous {
			return "", fmt.Errorf("第 %d 条审计事件的前序摘要不一致", i+1)
		}
		expected := AuditDigest(previous, event.JobID, event.EventType, event.ActorID, event.Payload, event.OccurredAt)
		if event.Digest != expected {
			return "", fmt.Errorf("第 %d 条审计事件摘要无效", i+1)
		}
		previous = event.Digest
	}
	return previous, nil
}
