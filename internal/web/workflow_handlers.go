package web

import (
	"net/http"
	"strconv"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

type revisionRequest struct {
	ParentRevisionID string           `json:"parentRevisionID"`
	AuthorID         string           `json:"authorID"`
	ChangeNote       string           `json:"changeNote"`
	ExpectedVersion  int64            `json:"expectedVersion"`
	IdempotencyKey   string           `json:"idempotencyKey"`
	Segments         []domain.Segment `json:"segments"`
	PreflightDigest  string           `json:"preflightDigest"`
	PreflightVersion int64            `json:"preflightVersion"`
}

func (a *API) HandleRevisions(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	var input revisionRequest
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	revision, err := a.service.SubmitRevision(r.Context(), application.SubmitRevisionCommand{JobID: jobID, ParentRevisionID: input.ParentRevisionID, AuthorID: input.AuthorID, ChangeNote: input.ChangeNote, ExpectedVersion: input.ExpectedVersion, IdempotencyKey: input.IdempotencyKey, Segments: input.Segments, PreflightDigest: input.PreflightDigest, PreflightVersion: input.PreflightVersion})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, revision)
}

func (a *API) HandleRevisionPreflight(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in revisionRequest
	if err = decodeJSON(w, r, &in); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.service.PreflightRevision(r.Context(), application.SubmitRevisionCommand{JobID: jobID, ParentRevisionID: in.ParentRevisionID, AuthorID: in.AuthorID, ChangeNote: in.ChangeNote, ExpectedVersion: in.ExpectedVersion, Segments: in.Segments})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

type qualityRequest struct {
	RevisionID      string `json:"revisionID"`
	ActorID         string `json:"actorID"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

func (a *API) HandleQualityCheck(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	var input qualityRequest
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	items, err := a.service.RunQuality(r.Context(), application.RunQualityCommand{JobID: jobID, RevisionID: input.RevisionID, ActorID: input.ActorID, ExpectedVersion: input.ExpectedVersion, IdempotencyKey: input.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"items": items})
}

type dispositionRequest struct {
	Disposition     domain.Disposition `json:"disposition"`
	Rationale       string             `json:"rationale"`
	ActorID         string             `json:"actorID"`
	ExpectedVersion int64              `json:"expectedVersion"`
	IdempotencyKey  string             `json:"idempotencyKey"`
}

func (a *API) HandleFindingDisposition(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	findingID, err := requiredPath(r, "findingID")
	if err != nil {
		writeError(w, err)
		return
	}
	var input dispositionRequest
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	finding, err := a.service.DispositionFinding(r.Context(), application.DispositionCommand{JobID: jobID, FindingID: findingID, Disposition: input.Disposition, Rationale: input.Rationale, ActorID: input.ActorID, ExpectedVersion: input.ExpectedVersion, IdempotencyKey: input.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, finding)
}

type batchDispositionRequest struct {
	Items           []application.BatchDispositionItem `json:"items"`
	ActorID         string                             `json:"actorID"`
	ExpectedVersion int64                              `json:"expectedVersion"`
	IdempotencyKey  string                             `json:"idempotencyKey"`
}

func (a *API) HandleFindingBatch(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	var in batchDispositionRequest
	if err = decodeJSON(w, r, &in); err != nil {
		writeError(w, err)
		return
	}
	v, err := a.service.BatchDisposition(r.Context(), application.BatchDispositionCommand{JobID: jobID, Items: in.Items, ActorID: in.ActorID, ExpectedVersion: in.ExpectedVersion, IdempotencyKey: in.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": v})
}

type reviewRequest struct {
	RevisionID        string   `json:"revisionID"`
	ReviewerID        string   `json:"reviewerID"`
	SampledSegmentIDs []string `json:"sampledSegmentIDs"`
	Decision          string   `json:"decision"`
	Comment           string   `json:"comment"`
	ExpectedVersion   int64    `json:"expectedVersion"`
	IdempotencyKey    string   `json:"idempotencyKey"`
}

func (a *API) HandleReview(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	var input reviewRequest
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	review, err := a.service.Review(r.Context(), application.ReviewCommand{JobID: jobID, RevisionID: input.RevisionID, ReviewerID: input.ReviewerID, SampledSegmentIDs: input.SampledSegmentIDs, Decision: input.Decision, Comment: input.Comment, ExpectedVersion: input.ExpectedVersion, IdempotencyKey: input.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

type freezeRequest struct {
	RevisionID         string `json:"revisionID"`
	ActorID            string `json:"actorID"`
	ExpectedVersion    int64  `json:"expectedVersion"`
	IdempotencyKey     string `json:"idempotencyKey"`
	ConfirmationDigest string `json:"confirmationDigest"`
}

func (a *API) HandleFreeze(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	var input freezeRequest
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	artifact, err := a.service.Freeze(r.Context(), application.FreezeCommand{JobID: jobID, RevisionID: input.RevisionID, ActorID: input.ActorID, ExpectedVersion: input.ExpectedVersion, IdempotencyKey: input.IdempotencyKey, ConfirmationDigest: input.ConfirmationDigest})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func (a *API) HandleSampling(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	revision := r.URL.Query().Get("revisionID")
	v, err := a.service.Sampling(r.Context(), jobID, revision)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (a *API) HandleFreezePreview(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	rev := r.URL.Query().Get("revisionID")
	expected, err := strconv.ParseInt(r.URL.Query().Get("expectedVersion"), 10, 64)
	if r.Method == http.MethodPost && r.ContentLength > 0 {
		var in struct {
			RevisionID      string `json:"revisionID"`
			ExpectedVersion int64  `json:"expectedVersion"`
		}
		if e := decodeJSON(w, r, &in); e != nil {
			writeError(w, e)
			return
		}
		rev = in.RevisionID
		expected = in.ExpectedVersion
		err = nil
	}
	if err != nil {
		writeError(w, domain.NewError(domain.CodeValidation, "expectedVersion 无效"))
		return
	}
	v, err := a.service.FreezePreview(r.Context(), jobID, rev, expected)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
