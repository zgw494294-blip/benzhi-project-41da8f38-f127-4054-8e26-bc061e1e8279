package web_test

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	store "benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/store/sqlite"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/web"
)

func TestIndexAndStrictJSON(t *testing.T) {
	repository, err := store.Open(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	handler := web.New(application.NewService(repository))
	request := httptest.NewRequest("GET", "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "<body>") {
		t.Fatalf("工作台 HTML 不完整: %d", response.Code)
	}
	request = httptest.NewRequest("POST", "/api/jobs", strings.NewReader(`{"title":"a","unknown":true}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 400 || !strings.Contains(response.Body.String(), "JSON 请求无效") {
		t.Fatalf("严格 JSON 未拒绝未知字段: %d %s", response.Code, response.Body.String())
	}
}

func TestFullHTTPFlow(t *testing.T) {
	repository, err := store.Open(filepath.Join(t.TempDir(), "flow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	server := httptest.NewServer(web.New(application.NewService(repository)))
	defer server.Close()
	if err := web.RunSelfcheck(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
}
