package application

import "benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"

type CreateJobCommand struct {
	Title, MediaRef, SourceLanguage, TargetLanguage, ProfileCode string
	DurationMillis                                               int64
	ActorID, IdempotencyKey                                      string
}

type SubmitRevisionCommand struct {
	JobID, ParentRevisionID, AuthorID, ChangeNote, IdempotencyKey string
	PreflightDigest                                               string
	PreflightVersion                                              int64
	ExpectedVersion                                               int64
	Segments                                                      []domain.Segment
}

type RunQualityCommand struct {
	JobID, RevisionID, ActorID, IdempotencyKey string
	ExpectedVersion                            int64
}

type JobQueueFilter struct {
	Keyword, Status, ProfileCode, SourceLanguage, TargetLanguage string
	Limit, Offset                                                int
}
type JobQueue struct {
	Items        []domain.CaptionJob      `json:"items"`
	Total        int                      `json:"total"`
	StatusCounts map[domain.JobStatus]int `json:"statusCounts"`
	Limit        int                      `json:"limit"`
	Offset       int                      `json:"offset"`
}
type RevisionPreflight struct {
	JobID            string        `json:"jobID"`
	ParentRevisionID string        `json:"parentRevisionID"`
	ContentDigest    string        `json:"contentDigest"`
	ExpectedVersion  int64         `json:"expectedVersion"`
	Sequence         int           `json:"sequence"`
	Summary          ChangeSummary `json:"summary"`
}
type ChangeSummary struct {
	Added          int    `json:"added"`
	Removed        int    `json:"removed"`
	TextChanged    int    `json:"textChanged"`
	TimingChanged  int    `json:"timingChanged"`
	SpeakerChanged int    `json:"speakerChanged"`
	Digest         string `json:"digest"`
}
type BatchDispositionItem struct {
	FindingID   string             `json:"findingID"`
	Disposition domain.Disposition `json:"disposition"`
	Rationale   string             `json:"rationale"`
}
type BatchDispositionCommand struct {
	JobID, ActorID, IdempotencyKey string
	ExpectedVersion                int64
	Items                          []BatchDispositionItem
}
type SamplingSuggestion struct {
	RevisionID    string   `json:"revisionID"`
	SegmentIDs    []string `json:"segmentIDs"`
	Minimum       int      `json:"minimum"`
	CoveredMillis int64    `json:"coveredMillis"`
	TotalMillis   int64    `json:"totalMillis"`
}
type FreezePreview struct {
	JobID           string `json:"jobID"`
	RevisionID      string `json:"revisionID"`
	ExpectedVersion int64  `json:"expectedVersion"`
	RevisionDigest  string `json:"revisionDigest"`
	AuditHeadDigest string `json:"auditHeadDigest"`
	SummaryDigest   string `json:"summaryDigest"`
	Reviewer        string `json:"reviewer"`
	RuleVersion     string `json:"ruleVersion"`
	Unresolved      int    `json:"unresolved"`
}

type DispositionCommand struct {
	JobID, FindingID, ActorID, Rationale, IdempotencyKey string
	Disposition                                          domain.Disposition
	ExpectedVersion                                      int64
}

type ReviewCommand struct {
	JobID, RevisionID, ReviewerID, Decision, Comment, IdempotencyKey string
	SampledSegmentIDs                                                []string
	ExpectedVersion                                                  int64
}

type FreezeCommand struct {
	JobID, RevisionID, ActorID, IdempotencyKey, ConfirmationDigest string
	ExpectedVersion                                                int64
}

type JobDetail struct {
	Job       domain.CaptionJob        `json:"job"`
	Revisions []domain.CaptionRevision `json:"revisions"`
	Findings  []domain.QualityFinding  `json:"findings"`
	Reviews   []domain.ReviewDecision  `json:"reviews"`
	Artifact  *domain.ReleaseArtifact  `json:"artifact,omitempty"`
}

type SegmentChange struct {
	SegmentID string          `json:"segmentID"`
	Kind      string          `json:"kind"`
	Before    *domain.Segment `json:"before,omitempty"`
	After     *domain.Segment `json:"after,omitempty"`
}

type RevisionDiff struct {
	FromRevisionID   string          `json:"fromRevisionID"`
	ToRevisionID     string          `json:"toRevisionID"`
	Changes          []SegmentChange `json:"changes"`
	FromFindingCount int             `json:"fromFindingCount"`
	ToFindingCount   int             `json:"toFindingCount"`
	FindingChanges   []FindingChange `json:"findingChanges,omitempty"`
}
type FindingChange struct {
	RuleCode  string `json:"ruleCode"`
	SegmentID string `json:"segmentID"`
	Kind      string `json:"kind"`
	Severity  string `json:"severity"`
}
