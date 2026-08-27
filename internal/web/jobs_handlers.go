package web

import (
	"net/http"
	"strconv"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

type createJobRequest struct {
	Title          string `json:"title"`
	MediaRef       string `json:"mediaRef"`
	DurationMillis int64  `json:"durationMillis"`
	SourceLanguage string `json:"sourceLanguage"`
	TargetLanguage string `json:"targetLanguage"`
	ProfileCode    string `json:"profileCode"`
	ActorID        string `json:"actorID"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (a *API) HandleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		limit, lerr := strconv.Atoi(q.Get("limit"))
		offset, oerr := strconv.Atoi(q.Get("offset"))
		if (q.Get("limit") != "" && lerr != nil) || (q.Get("offset") != "" && oerr != nil) {
			writeError(w, domain.NewError(domain.CodeValidation, "分页参数必须为数字"))
			return
		}
		queue, err := a.service.QueryWorkQueue(r.Context(), application.JobQueueFilter{Keyword: q.Get("keyword"), Status: q.Get("status"), ProfileCode: q.Get("profileCode"), SourceLanguage: q.Get("sourceLanguage"), TargetLanguage: q.Get("targetLanguage"), Limit: limit, Offset: offset})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, queue)
		return
	}
	var input createJobRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	job, err := a.service.CreateJob(r.Context(), application.CreateJobCommand{Title: input.Title, MediaRef: input.MediaRef, DurationMillis: input.DurationMillis, SourceLanguage: input.SourceLanguage, TargetLanguage: input.TargetLanguage, ProfileCode: input.ProfileCode, ActorID: input.ActorID, IdempotencyKey: input.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (a *API) HandleJobDetail(w http.ResponseWriter, r *http.Request) {
	id, err := requiredPath(r, "jobID")
	if err != nil {
		writeError(w, err)
		return
	}
	detail, err := a.service.JobDetail(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
