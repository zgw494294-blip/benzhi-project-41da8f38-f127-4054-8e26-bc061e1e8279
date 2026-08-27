package domain

import "time"

type JobStatus string

const (
	StatusDraft       JobStatus = "draft"
	StatusRemediating JobStatus = "remediating"
	StatusReviewReady JobStatus = "review_ready"
	StatusApproved    JobStatus = "approved"
	StatusFrozen      JobStatus = "frozen"
)

type CaptionJob struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	MediaRef          string    `json:"mediaRef"`
	DurationMillis    int64     `json:"durationMillis"`
	SourceLanguage    string    `json:"sourceLanguage"`
	TargetLanguage    string    `json:"targetLanguage"`
	ProfileCode       string    `json:"profileCode"`
	Status            JobStatus `json:"status"`
	CurrentRevisionID string    `json:"currentRevisionID,omitempty"`
	Version           int64     `json:"version"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type Segment struct {
	ID          string `json:"id"`
	StartMillis int64  `json:"startMillis"`
	EndMillis   int64  `json:"endMillis"`
	Speaker     string `json:"speaker"`
	Text        string `json:"text"`
}

type CaptionRevision struct {
	ID               string    `json:"id"`
	JobID            string    `json:"jobID"`
	ParentRevisionID string    `json:"parentRevisionID,omitempty"`
	Sequence         int       `json:"sequence"`
	AuthorID         string    `json:"authorID"`
	ChangeNote       string    `json:"changeNote"`
	Segments         []Segment `json:"segments"`
	ContentDigest    string    `json:"contentDigest"`
	SubmittedAt      time.Time `json:"submittedAt"`
}

type Disposition string

const (
	DispositionOpen      Disposition = "open"
	DispositionConfirmed Disposition = "confirmed"
	DispositionFalse     Disposition = "false_positive"
	DispositionFix       Disposition = "needs_fix"
)

type QualityFinding struct {
	ID          string      `json:"id"`
	JobID       string      `json:"jobID"`
	RevisionID  string      `json:"revisionID"`
	RuleCode    string      `json:"ruleCode"`
	Severity    string      `json:"severity"`
	SegmentIDs  []string    `json:"segmentIDs"`
	Message     string      `json:"message"`
	Disposition Disposition `json:"disposition"`
	Rationale   string      `json:"rationale,omitempty"`
	ResolvedBy  string      `json:"resolvedBy,omitempty"`
	ResolvedAt  *time.Time  `json:"resolvedAt,omitempty"`
}

type ReviewDecision struct {
	ID                string    `json:"id"`
	JobID             string    `json:"jobID"`
	RevisionID        string    `json:"revisionID"`
	ReviewerID        string    `json:"reviewerID"`
	SampledSegmentIDs []string  `json:"sampledSegmentIDs"`
	Decision          string    `json:"decision"`
	Comment           string    `json:"comment"`
	DecidedAt         time.Time `json:"decidedAt"`
}

type ReleaseArtifact struct {
	ID                 string    `json:"id"`
	JobID              string    `json:"jobID"`
	RevisionID         string    `json:"revisionID"`
	CredentialNumber   string    `json:"credentialNumber"`
	WebVTT             string    `json:"webvtt,omitempty"`
	Manifest           string    `json:"manifest,omitempty"`
	WebVTTDigest       string    `json:"webvttDigest"`
	ManifestDigest     string    `json:"manifestDigest"`
	AuditHeadDigest    string    `json:"auditHeadDigest"`
	FrozenBy           string    `json:"frozenBy"`
	FrozenAt           time.Time `json:"frozenAt"`
	VerificationStatus string    `json:"verificationStatus"`
}

type AuditEvent struct {
	ID         int64     `json:"id"`
	JobID      string    `json:"jobID"`
	EventType  string    `json:"eventType"`
	ActorID    string    `json:"actorID"`
	Payload    string    `json:"payload"`
	PrevDigest string    `json:"prevDigest"`
	Digest     string    `json:"digest"`
	OccurredAt time.Time `json:"occurredAt"`
}

type VerificationRecord struct {
	ID              string    `json:"id"`
	JobID           string    `json:"jobID"`
	ArtifactID      string    `json:"artifactID"`
	VerifierID      string    `json:"verifierID"`
	Valid           bool      `json:"valid"`
	Reasons         []string  `json:"reasons"`
	WebVTTDigest    string    `json:"webvttDigest"`
	ManifestDigest  string    `json:"manifestDigest"`
	AuditHeadDigest string    `json:"auditHeadDigest"`
	CreatedAt       time.Time `json:"createdAt"`
}
