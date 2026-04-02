package service

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"baby-record-server/internal/auth"
	"baby-record-server/internal/repository"
	"baby-record-server/internal/wechat"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	loc := time.FixedZone("CST", 8*3600)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	repo := repository.New(db)
	if err := repo.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return New(
		repo,
		wechat.New("", "", true),
		auth.NewManager("test-secret", "test-issuer", time.Hour),
		loc,
		false,
	)
}

func TestFamilyAndRecordFlow(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	login1, err := svc.Login(ctx, LoginInput{Code: "user-1"})
	if err != nil {
		t.Fatalf("login1: %v", err)
	}
	user1 := login1.User.ID

	family, err := svc.CreateFamily(ctx, user1, CreateFamilyInput{Name: "开心一家"})
	if err != nil {
		t.Fatalf("create family: %v", err)
	}
	if family.Role != "owner" {
		t.Fatalf("unexpected role: %s", family.Role)
	}

	baby, err := svc.CreateBaby(ctx, user1, UpsertBabyInput{
		Name:        "小满",
		Gender:      "girl",
		Birthday:    "2024-10-01",
		FeedingMode: "mixed",
	})
	if err != nil {
		t.Fatalf("create baby: %v", err)
	}
	if baby.Name != "小满" {
		t.Fatalf("unexpected baby name: %s", baby.Name)
	}

	feed, err := svc.CreateRecord(ctx, user1, UpsertRecordInput{
		Type:       "feed",
		Subtype:    "formula",
		OccurredAt: "2026-04-01T09:00:00+08:00",
		Amount:     floatPtr(120),
		Unit:       "ml",
	})
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}

	if _, err := svc.CreateRecord(ctx, user1, UpsertRecordInput{
		Type:    "sleep",
		StartAt: "2026-04-01T10:00:00+08:00",
		EndAt:   "2026-04-01T11:30:00+08:00",
	}); err != nil {
		t.Fatalf("create sleep: %v", err)
	}

	login2, err := svc.Login(ctx, LoginInput{Code: "user-2"})
	if err != nil {
		t.Fatalf("login2: %v", err)
	}
	user2 := login2.User.ID

	joinRequest, err := svc.JoinFamily(ctx, user2, JoinFamilyInput{Code: strings.ToLower(family.InviteCode)})
	if err != nil {
		t.Fatalf("submit join request: %v", err)
	}
	if joinRequest.Status != "pending" {
		t.Fatalf("unexpected join request status: %s", joinRequest.Status)
	}

	requests, err := svc.ListJoinRequests(ctx, user1)
	if err != nil {
		t.Fatalf("list join requests: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("unexpected join request count: %d", len(requests))
	}

	if _, err := svc.ReviewJoinRequest(ctx, user1, ReviewJoinRequestInput{
		RequestID: requests[0].ID,
		Approve:   true,
	}); err != nil {
		t.Fatalf("approve join request: %v", err)
	}

	if _, err := svc.UpdateRecord(ctx, user2, feed.ID, UpsertRecordInput{
		Type:       "feed",
		Subtype:    "formula",
		OccurredAt: "2026-04-01T09:00:00+08:00",
		Amount:     floatPtr(90),
		Unit:       "ml",
	}); err == nil {
		t.Fatalf("member should not update owner record")
	}

	dashboard, err := svc.GetDashboard(ctx, user1, "2026-04-01")
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.Summary.FeedCount != 1 || dashboard.Summary.SleepDurationMin != 90 {
		t.Fatalf("unexpected summary: %+v", dashboard.Summary)
	}

	if !regexp.MustCompile(`^[A-Z0-9]{6}$`).MatchString(family.InviteCode) {
		t.Fatalf("unexpected invite code format: %s", family.InviteCode)
	}
}

func TestStatsAggregation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	login, err := svc.Login(ctx, LoginInput{Code: "stats-user"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	userID := login.User.ID

	if _, err := svc.CreateFamily(ctx, userID, CreateFamilyInput{Name: "统计家庭"}); err != nil {
		t.Fatalf("create family: %v", err)
	}
	if _, err := svc.CreateBaby(ctx, userID, UpsertBabyInput{
		Name:     "阿福",
		Gender:   "boy",
		Birthday: "2024-09-01",
	}); err != nil {
		t.Fatalf("create baby: %v", err)
	}

	for _, payload := range []UpsertRecordInput{
		{Type: "feed", Subtype: "formula", OccurredAt: time.Now().Add(-24 * time.Hour).Format(time.RFC3339), Amount: floatPtr(100), Unit: "ml"},
		{Type: "diaper", Subtype: "mixed", OccurredAt: time.Now().Format(time.RFC3339)},
		{Type: "growth", OccurredAt: time.Now().Format(time.RFC3339), WeightKg: floatPtr(7.4)},
	} {
		if _, err := svc.CreateRecord(ctx, userID, payload); err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	stats, err := svc.GetStats(ctx, userID, StatsQueryInput{Range: "7d"})
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if len(stats.Series) != 7 {
		t.Fatalf("unexpected series len: %d", len(stats.Series))
	}
	if len(stats.GrowthTrend) != 1 {
		t.Fatalf("unexpected growth len: %d", len(stats.GrowthTrend))
	}

	customStats, err := svc.GetStats(ctx, userID, StatsQueryInput{
		StartDate: time.Now().Add(-24 * time.Hour).Format("2006-01-02"),
		EndDate:   time.Now().Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("get custom stats: %v", err)
	}
	if customStats.Range != "custom" {
		t.Fatalf("unexpected range: %s", customStats.Range)
	}
	if len(customStats.Series) != 2 {
		t.Fatalf("unexpected custom series len: %d", len(customStats.Series))
	}
}
