package evidence

import (
	"encoding/json"
	"fmt"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

type VerificationResult struct {
	Valid        bool     `json:"valid"`
	Reasons      []string `json:"reasons"`
	WebVTTDigest string   `json:"webvttDigest"`
	ManifestHash string   `json:"manifestDigest"`
	AuditHead    string   `json:"auditHeadDigest"`
}

func VerifyRelease(revision domain.CaptionRevision, artifact domain.ReleaseArtifact, events []domain.AuditEvent) VerificationResult {
	result := VerificationResult{Valid: true, Reasons: []string{}}
	vtt, digest := BuildWebVTT(revision)
	result.WebVTTDigest = digest
	if string(vtt) != artifact.WebVTT || digest != artifact.WebVTTDigest {
		result.Reasons = append(result.Reasons, "WebVTT 内容或摘要不一致")
	}
	result.ManifestHash = SHA256([]byte(artifact.Manifest))
	if result.ManifestHash != artifact.ManifestDigest {
		result.Reasons = append(result.Reasons, "冻结清单摘要不一致")
	}
	var manifest Manifest
	if err := json.Unmarshal([]byte(artifact.Manifest), &manifest); err != nil {
		result.Reasons = append(result.Reasons, "冻结清单无法解析")
	} else {
		if manifest.CredentialNumber != artifact.CredentialNumber {
			result.Reasons = append(result.Reasons, "凭据编号与冻结清单不一致")
		}
		if manifest.RevisionID != revision.ID || manifest.RevisionDigest != revision.ContentDigest {
			result.Reasons = append(result.Reasons, "修订身份或摘要不一致")
		}
	}
	head, err := VerifyAudit(events)
	if err != nil {
		result.Reasons = append(result.Reasons, fmt.Sprintf("审计链验证失败：%v", err))
	}
	result.AuditHead = head
	// 清单记录的是冻结发生前的链头；发布事件本身随后进入审计链。
	if manifest.AuditHeadDigest != artifact.AuditHeadDigest {
		result.Reasons = append(result.Reasons, "冻结清单中的审计链头不一致")
	}
	result.Valid = len(result.Reasons) == 0
	return result
}
