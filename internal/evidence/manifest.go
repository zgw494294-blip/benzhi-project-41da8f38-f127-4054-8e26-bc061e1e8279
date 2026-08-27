package evidence

import (
	"encoding/json"
	"fmt"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

type Manifest struct {
	SchemaVersion    string `json:"schemaVersion"`
	JobID            string `json:"jobID"`
	RevisionID       string `json:"revisionID"`
	RevisionDigest   string `json:"revisionDigest"`
	WebVTTDigest     string `json:"webvttDigest"`
	RuleVersion      string `json:"ruleVersion"`
	AuditHeadDigest  string `json:"auditHeadDigest"`
	CredentialNumber string `json:"credentialNumber"`
	FrozenBy         string `json:"frozenBy"`
	FrozenAt         string `json:"frozenAt"`
}

func CredentialNumber(jobID, revisionID, webvttDigest string) string {
	digest := SHA256([]byte(jobID + "\n" + revisionID + "\n" + webvttDigest))
	return fmt.Sprintf("ACR-%s-%s", time.Now().UTC().Format("20060102"), digest[:16])
}

func BuildManifest(job domain.CaptionJob, revision domain.CaptionRevision, actor, auditHead, credential string, frozenAt time.Time) ([]byte, string, error) {
	_, vttDigest := BuildWebVTT(revision)
	manifest := Manifest{
		SchemaVersion: "caption-release-manifest-v1", JobID: job.ID, RevisionID: revision.ID,
		RevisionDigest: revision.ContentDigest, WebVTTDigest: vttDigest, RuleVersion: domain.RuleVersion,
		AuditHeadDigest: auditHead, CredentialNumber: credential, FrozenBy: actor, FrozenAt: frozenAt.UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, "", err
	}
	data = append(data, '\n')
	return data, SHA256(data), nil
}
