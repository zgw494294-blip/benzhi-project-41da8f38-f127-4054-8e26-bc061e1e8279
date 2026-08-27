package application_test

import (
	"context"
	"path/filepath"
	"testing"

	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/application"
	"benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/domain"
	store "benzhi-project-41da8f38-f127-4054-8e26-bc061e1e8279/internal/store/sqlite"
)

func testService(t *testing.T) (*application.Service, *store.Store) {
	t.Helper()
	repository, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	return application.NewService(repository), repository
}

func TestWorkflowIdempotencyVersioningAndFreeze(t *testing.T) {
	ctx := context.Background()
	service, repository := testService(t)
	create := application.CreateJobCommand{Title: "测试节目", MediaRef: "media://test", DurationMillis: 10000, SourceLanguage: "zh-CN", TargetLanguage: "zh-CN", ProfileCode: "P1", ActorID: "editor", IdempotencyKey: "create-key-001"}
	job, err := service.CreateJob(ctx, create)
	if err != nil {
		t.Fatal(err)
	}
	again, err := service.CreateJob(ctx, create)
	if err != nil || again.ID != job.ID {
		t.Fatalf("建档幂等失败: %v", err)
	}
	segments := []domain.Segment{{ID: "s1", StartMillis: 0, EndMillis: 1500, Text: "[音乐]"}, {ID: "s2", StartMillis: 1700, EndMillis: 4000, Text: "欢迎收看节目"}}
	revision, err := service.SubmitRevision(ctx, application.SubmitRevisionCommand{JobID: job.ID, AuthorID: "editor", ChangeNote: "首版", ExpectedVersion: 1, IdempotencyKey: "revision-key-001", Segments: segments})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RunQuality(ctx, application.RunQualityCommand{JobID: job.ID, RevisionID: revision.ID, ActorID: "editor", ExpectedVersion: 1, IdempotencyKey: "wrong-version-001"}); domain.ErrorCodeOf(err) != domain.CodeConflict {
		t.Fatalf("应检测版本冲突: %v", err)
	}
	findings, err := service.RunQuality(ctx, application.RunQualityCommand{JobID: job.ID, RevisionID: revision.ID, ActorID: "editor", ExpectedVersion: 2, IdempotencyKey: "quality-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	version := int64(3)
	for i, f := range findings {
		_, err = service.DispositionFinding(ctx, application.DispositionCommand{JobID: job.ID, FindingID: f.ID, ActorID: "editor", Disposition: domain.DispositionFalse, Rationale: "符合测试规范", ExpectedVersion: version, IdempotencyKey: "finding-key-00" + string(rune('0'+i))})
		if err != nil {
			t.Fatal(err)
		}
		version++
	}
	if _, err := service.Review(ctx, application.ReviewCommand{JobID: job.ID, RevisionID: revision.ID, ReviewerID: "editor", SampledSegmentIDs: []string{"s1"}, Decision: "approve", Comment: "自审", ExpectedVersion: version, IdempotencyKey: "self-review-001"}); domain.ErrorCodeOf(err) != domain.CodeForbidden {
		t.Fatalf("应拒绝自审: %v", err)
	}
	_, err = service.Review(ctx, application.ReviewCommand{JobID: job.ID, RevisionID: revision.ID, ReviewerID: "reviewer", SampledSegmentIDs: []string{"s1", "s2"}, Decision: "approve", Comment: "抽检通过", ExpectedVersion: version, IdempotencyKey: "review-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	version++
	artifact, err := service.Freeze(ctx, application.FreezeCommand{JobID: job.ID, RevisionID: revision.ID, ActorID: "publisher", ExpectedVersion: version, IdempotencyKey: "freeze-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.CredentialNumber == "" || artifact.ManifestDigest == "" {
		t.Fatal("发布证据不完整")
	}
	verification, err := service.Verify(ctx, job.ID)
	if err != nil || !verification.Valid {
		t.Fatalf("发布验证失败: %#v %v", verification, err)
	}
	if err := repository.CheckIntegrity(ctx); err != nil {
		t.Fatalf("存储或审计完整性失败: %v", err)
	}
}

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persistent.db")
	repository, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repository)
	job, err := service.CreateJob(context.Background(), application.CreateJobCommand{Title: "持久任务", MediaRef: "media://persist", DurationMillis: 1000, SourceLanguage: "zh", TargetLanguage: "zh", ProfileCode: "P", ActorID: "editor", IdempotencyKey: "persist-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	repository.Close()
	repository, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	service = application.NewService(repository)
	detail, err := service.JobDetail(context.Background(), job.ID)
	if err != nil || detail.Job.Title != "持久任务" {
		t.Fatalf("重启后数据丢失: %#v %v", detail, err)
	}
}
