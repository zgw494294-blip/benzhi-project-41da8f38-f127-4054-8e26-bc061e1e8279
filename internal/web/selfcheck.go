package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
)

type checkClient struct {
	base   string
	client *http.Client
}

func RunSelfcheck(ctx context.Context, baseURL string) error {
	c := &checkClient{base: baseURL, client: &http.Client{Timeout: 5 * time.Second}}
	if err := c.get(ctx, "/", nil); err != nil {
		return fmt.Errorf("加载工作台: %w", err)
	}
	var job domain.CaptionJob
	if err := c.json(ctx, http.MethodPost, "/api/jobs", map[string]any{"title": "自检无障碍节目", "mediaRef": "media://selfcheck/reference", "durationMillis": 10000, "sourceLanguage": "zh-CN", "targetLanguage": "zh-CN", "profileCode": "PUBLIC-CULTURE-V1", "actorID": "editor-selfcheck", "idempotencyKey": "selfcheck-create-001"}, http.StatusCreated, &job); err != nil {
		return fmt.Errorf("建档: %w", err)
	}
	segments := []domain.Segment{{ID: "s1", StartMillis: 0, EndMillis: 500, Speaker: "环境声", Text: "[音乐]"}, {ID: "s2", StartMillis: 700, EndMillis: 2700, Speaker: "主持人", Text: "欢迎收看公共文化节目。"}}
	var revision domain.CaptionRevision
	if err := c.json(ctx, http.MethodPost, "/api/jobs/"+job.ID+"/revisions", map[string]any{"parentRevisionID": "", "authorID": "editor-selfcheck", "changeNote": "首个可审校版本", "expectedVersion": int64(1), "idempotencyKey": "selfcheck-revision-001", "segments": segments}, http.StatusCreated, &revision); err != nil {
		return fmt.Errorf("提交修订: %w", err)
	}
	var quality struct {
		Items []domain.QualityFinding `json:"items"`
	}
	if err := c.json(ctx, http.MethodPost, "/api/jobs/"+job.ID+"/quality-checks", map[string]any{"revisionID": revision.ID, "actorID": "editor-selfcheck", "expectedVersion": int64(2), "idempotencyKey": "selfcheck-quality-001"}, http.StatusCreated, &quality); err != nil {
		return fmt.Errorf("质量检查: %w", err)
	}
	version := int64(3)
	for i, finding := range quality.Items {
		var updated domain.QualityFinding
		if err := c.json(ctx, http.MethodPatch, "/api/jobs/"+job.ID+"/findings/"+finding.ID, map[string]any{"disposition": "false_positive", "rationale": "自检样本的短声音提示符合节目发布规范", "actorID": "editor-selfcheck", "expectedVersion": version, "idempotencyKey": fmt.Sprintf("selfcheck-finding-%03d", i)}, http.StatusOK, &updated); err != nil {
			return fmt.Errorf("处置问题: %w", err)
		}
		version++
	}
	var review domain.ReviewDecision
	if err := c.json(ctx, http.MethodPost, "/api/jobs/"+job.ID+"/reviews", map[string]any{"revisionID": revision.ID, "reviewerID": "reviewer-selfcheck", "sampledSegmentIDs": []string{"s1", "s2"}, "decision": "approve", "comment": "时间轴、文本和声音提示抽检通过", "expectedVersion": version, "idempotencyKey": "selfcheck-review-001"}, http.StatusCreated, &review); err != nil {
		return fmt.Errorf("独立复核: %w", err)
	}
	version++
	var artifact domain.ReleaseArtifact
	if err := c.json(ctx, http.MethodPost, "/api/jobs/"+job.ID+"/freeze", map[string]any{"revisionID": revision.ID, "actorID": "publisher-selfcheck", "expectedVersion": version, "idempotencyKey": "selfcheck-freeze-001"}, http.StatusCreated, &artifact); err != nil {
		return fmt.Errorf("冻结发布: %w", err)
	}
	var verification struct {
		Valid   bool     `json:"valid"`
		Reasons []string `json:"reasons"`
	}
	if err := c.json(ctx, http.MethodPost, "/api/jobs/"+job.ID+"/verify", map[string]any{}, http.StatusOK, &verification); err != nil {
		return fmt.Errorf("验证发布包: %w", err)
	}
	if !verification.Valid {
		return fmt.Errorf("发布包验证失败: %v", verification.Reasons)
	}
	var audit struct {
		Items []domain.AuditEvent `json:"items"`
	}
	if err := c.get(ctx, "/api/jobs/"+job.ID+"/audit", &audit); err != nil {
		return fmt.Errorf("读取审计轨迹: %w", err)
	}
	if len(audit.Items) < 5 {
		return fmt.Errorf("审计事件数量不足: %d", len(audit.Items))
	}
	var vtt string
	if err := c.get(ctx, "/api/jobs/"+job.ID+"/artifact/captions.vtt", &vtt); err != nil {
		return fmt.Errorf("下载 WebVTT: %w", err)
	}
	if len(vtt) < 20 {
		return fmt.Errorf("WebVTT 内容不完整")
	}
	return nil
}

func (c *checkClient) json(ctx context.Context, method, path string, input any, want int, target any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != want {
		return fmt.Errorf("%s %s 返回 %d: %s", method, path, res.StatusCode, string(body))
	}
	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return err
		}
	}
	return nil
}
func (c *checkClient) get(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s 返回 %d: %s", path, res.StatusCode, string(body))
	}
	if target == nil {
		return nil
	}
	if text, ok := target.(*string); ok {
		*text = string(body)
		return nil
	}
	return json.Unmarshal(body, target)
}

var _ = application.JobDetail{}
