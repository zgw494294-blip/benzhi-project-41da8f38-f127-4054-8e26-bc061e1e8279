package web

import (
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"net/http"
	"strconv"
)

func (a *API) HandleRevisionDiff(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	diff, err := a.service.Diff(r.Context(), jobID, from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}
func (a *API) HandleAudit(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := a.service.Timeline(r.Context(), jobID, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (a *API) HandleVerify(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	if r.ContentLength > 0 {
		var input struct {
			VerifierID      string `json:"verifierID"`
			ExpectedVersion int64  `json:"expectedVersion"`
			IdempotencyKey  string `json:"idempotencyKey"`
		}
		if err = decodeJSON(w, r, &input); err != nil {
			writeError(w, err)
			return
		}
		if input.VerifierID != "" || input.IdempotencyKey != "" {
			rec, e := a.service.VerifyAndRecord(r.Context(), application.VerifyCommand{JobID: jobID, VerifierID: input.VerifierID, ExpectedVersion: input.ExpectedVersion, IdempotencyKey: input.IdempotencyKey})
			if e != nil {
				writeError(w, e)
				return
			}
			writeJSON(w, http.StatusOK, rec)
			return
		}
	}
	result, err := a.service.Verify(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) HandleVerificationHistory(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	v, err := a.service.VerificationHistory(r.Context(), jobID, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": v})
}
func (a *API) HandleArtifact(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	artifact, err := a.service.Artifact(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, artifact)
}
func (a *API) HandleWebVTT(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	artifact, err := a.service.Artifact(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="captions.vtt"`)
	_, _ = w.Write([]byte(artifact.WebVTT))
}
func (a *API) HandleManifest(w http.ResponseWriter, r *http.Request) {
	jobID, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	artifact, err := a.service.Artifact(r.Context(), jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="manifest.json"`)
	_, _ = w.Write([]byte(artifact.Manifest))
}
