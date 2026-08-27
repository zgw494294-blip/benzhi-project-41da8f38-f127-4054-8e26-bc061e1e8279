package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

//go:embed assets/*
var assets embed.FS

type API struct {
	service *application.Service
	mux     *http.ServeMux
}

func New(service *application.Service) *API {
	a := &API{service: service, mux: http.NewServeMux()}
	a.routes()
	return a
}

func (a *API) routes() {
	a.mux.HandleFunc("GET /", a.HandleIndex)
	static, _ := fs.Sub(assets, "assets")
	a.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(static))))
	a.mux.HandleFunc("GET /api/health", a.HandleHealth)
	a.mux.HandleFunc("GET /api/jobs", a.HandleJobs)
	a.mux.HandleFunc("POST /api/jobs", a.HandleJobs)
	a.mux.HandleFunc("GET /api/jobs/{jobID}", a.HandleJobDetail)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/revisions", a.HandleRevisions)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/revisions/preflight", a.HandleRevisionPreflight)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/diff", a.HandleRevisionDiff)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/quality-checks", a.HandleQualityCheck)
	a.mux.HandleFunc("PATCH /api/jobs/{jobID}/findings/{findingID}", a.HandleFindingDisposition)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/findings/batch", a.HandleFindingBatch)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/reviews", a.HandleReview)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/freeze", a.HandleFreeze)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/freeze-preview", a.HandleFreezePreview)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/freeze-preview", a.HandleFreezePreview)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/sampling", a.HandleSampling)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/sampling", a.HandleSampling)
	a.mux.HandleFunc("POST /api/jobs/{jobID}/verify", a.HandleVerify)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/verification-history", a.HandleVerificationHistory)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/audit", a.HandleAudit)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/artifact", a.HandleArtifact)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/artifact/captions.vtt", a.HandleWebVTT)
	a.mux.HandleFunc("GET /api/jobs/{jobID}/artifact/manifest.json", a.HandleManifest)
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	a.mux.ServeHTTP(w, r)
}

func (a *API) HandleIndex(w http.ResponseWriter, _ *http.Request) {
	data, _ := assets.ReadFile("assets/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
func (a *API) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return domain.NewError(domain.CodeValidation, "请求体不能为空")
		}
		return domain.NewError(domain.CodeValidation, "JSON 请求无效：%v", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.NewError(domain.CodeValidation, "JSON 请求只能包含一个对象")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := domain.ErrorCodeOf(err)
	switch code {
	case domain.CodeValidation:
		status = http.StatusBadRequest
	case domain.CodeNotFound:
		status = http.StatusNotFound
	case domain.CodeConflict, domain.CodeState:
		status = http.StatusConflict
	case domain.CodeForbidden:
		status = http.StatusForbidden
	}
	message := "服务内部错误"
	if code != "internal" {
		message = err.Error()
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": string(code), "message": message}})
}

func requiredPath(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.PathValue(name))
	if value == "" {
		return "", domain.NewError(domain.CodeValidation, "路径参数 %s 不能为空", name)
	}
	return value, nil
}
