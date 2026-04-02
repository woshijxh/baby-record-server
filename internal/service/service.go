package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"baby-record-server/internal/apperror"
	"baby-record-server/internal/auth"
	"baby-record-server/internal/model"
	"baby-record-server/internal/repository"
	"baby-record-server/internal/wechat"
)

type Service struct {
	repo             repository.Repository
	wechat           wechat.Client
	token            *auth.Manager
	location         *time.Location
	autoSeedDemoData bool
}

func New(repo repository.Repository, wechatClient wechat.Client, tokenManager *auth.Manager, location *time.Location, autoSeedDemoData bool) *Service {
	return &Service{
		repo:     repo,
		wechat:   wechatClient,
		token:    tokenManager,
		location: location,
		autoSeedDemoData: autoSeedDemoData,
	}
}

type LoginInput struct {
	Code      string
	Nickname  string
	AvatarURL string
}

type CreateFamilyInput struct {
	Name string
}

type JoinFamilyInput struct {
	Code string
}

type ReviewJoinRequestInput struct {
	RequestID uint64
	Approve   bool
}

type UpsertBabyInput struct {
	Name        string
	Gender      string
	Birthday    string
	AvatarURL   string
	FeedingMode string
	AllergyNote string
}

type UpsertRecordInput struct {
	Type                string   `json:"type"`
	Subtype             string   `json:"subtype"`
	OccurredAt          string   `json:"occurredAt"`
	StartAt             string   `json:"startAt"`
	EndAt               string   `json:"endAt"`
	Amount              *float64 `json:"amount"`
	Unit                string   `json:"unit"`
	DurationMin         *int     `json:"durationMin"`
	WeightKg            *float64 `json:"weightKg"`
	HeightCm            *float64 `json:"heightCm"`
	HeadCircumferenceCm *float64 `json:"headCircumferenceCm"`
	Note                string   `json:"note"`
}

type UserResponse struct {
	ID        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

type FamilyResponse struct {
	ID              uint64 `json:"id"`
	Name            string `json:"name"`
	Role            string `json:"role"`
	InviteCode      string `json:"inviteCode"`
	InviteExpiresAt string `json:"inviteExpiresAt"`
}

type JoinRequestResponse struct {
	ID         uint64 `json:"id"`
	FamilyID   uint64 `json:"familyId"`
	FamilyName string `json:"familyName"`
	Requester  string `json:"requester"`
	Status     string `json:"status"`
	InviteCode string `json:"inviteCode"`
	CreatedAt  string `json:"createdAt"`
	ReviewedAt string `json:"reviewedAt"`
}

type BabyResponse struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Birthday    string `json:"birthday"`
	AvatarURL   string `json:"avatarUrl"`
	FeedingMode string `json:"feedingMode"`
	AllergyNote string `json:"allergyNote"`
}

type LoginResponse struct {
	Token          string          `json:"token"`
	User           UserResponse    `json:"user"`
	Family         *FamilyResponse `json:"family"`
	Baby           *BabyResponse   `json:"baby"`
	NeedOnboarding bool            `json:"needOnboarding"`
}

type DashboardSummary struct {
	FeedCount        int `json:"feedCount"`
	SleepCount       int `json:"sleepCount"`
	SleepDurationMin int `json:"sleepDurationMin"`
	DiaperCount      int `json:"diaperCount"`
	GrowthCount      int `json:"growthCount"`
}

type RecordResponse struct {
	ID                  uint64   `json:"id"`
	Type                string   `json:"type"`
	Subtype             string   `json:"subtype"`
	OccurredAt          string   `json:"occurredAt"`
	StartAt             string   `json:"startAt"`
	EndAt               string   `json:"endAt"`
	Amount              *float64 `json:"amount,omitempty"`
	Unit                string   `json:"unit"`
	DurationMin         *int     `json:"durationMin,omitempty"`
	WeightKg            *float64 `json:"weightKg,omitempty"`
	HeightCm            *float64 `json:"heightCm,omitempty"`
	HeadCircumferenceCm *float64 `json:"headCircumferenceCm,omitempty"`
	Note                string   `json:"note"`
	CreatedBy           uint64   `json:"createdBy"`
	CanEdit             bool     `json:"canEdit"`
}

type DashboardResponse struct {
	Date          string           `json:"date"`
	Baby          *BabyResponse    `json:"baby"`
	Summary       DashboardSummary `json:"summary"`
	LatestGrowth  *RecordResponse  `json:"latestGrowth"`
	RecentRecords []RecordResponse `json:"recentRecords"`
}

type DailyStat struct {
	Date             string `json:"date"`
	FeedCount        int    `json:"feedCount"`
	SleepDurationMin int    `json:"sleepDurationMin"`
	DiaperCount      int    `json:"diaperCount"`
}

type GrowthPoint struct {
	Date                string   `json:"date"`
	WeightKg            *float64 `json:"weightKg,omitempty"`
	HeightCm            *float64 `json:"heightCm,omitempty"`
	HeadCircumferenceCm *float64 `json:"headCircumferenceCm,omitempty"`
}

type StatsResponse struct {
	Range       string           `json:"range"`
	StartDate   string           `json:"startDate"`
	EndDate     string           `json:"endDate"`
	Summary     DashboardSummary `json:"summary"`
	Series      []DailyStat      `json:"series"`
	GrowthTrend []GrowthPoint    `json:"growthTrend"`
}

type StatsQueryInput struct {
	Range     string
	StartDate string
	EndDate   string
}

type familyScope struct {
	Membership *repository.MembershipContext
	Baby       *model.Baby
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*LoginResponse, error) {
	if strings.TrimSpace(input.Code) == "" {
		return nil, apperror.BadRequest("code 不能为空", nil)
	}

	openID, err := s.wechat.CodeToSession(ctx, input.Code)
	if err != nil {
		return nil, apperror.Unauthorized("微信登录失败", err)
	}

	user, err := s.repo.UpsertUserByOpenID(ctx, openID, strings.TrimSpace(input.Nickname), strings.TrimSpace(input.AvatarURL))
	if err != nil {
		return nil, apperror.Internal("保存用户失败", err)
	}

	tokenString, err := s.token.Sign(user.ID)
	if err != nil {
		return nil, apperror.Internal("生成 token 失败", err)
	}

	resp := &LoginResponse{
		Token:          tokenString,
		User:           s.toUserResponse(user),
		NeedOnboarding: true,
	}

	scope, err := s.getFamilyScope(ctx, user.ID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return resp, nil
	}

	invite, err := s.repo.EnsureActiveInvite(ctx, scope.Membership.Family.ID, user.ID)
	if err != nil {
		return nil, apperror.Internal("获取邀请码失败", err)
	}
	resp.Family = s.toFamilyResponse(&scope.Membership.Family, &scope.Membership.Member, invite)
	resp.NeedOnboarding = scope.Baby == nil
	if scope.Baby != nil {
		resp.Baby = s.toBabyResponse(scope.Baby)
		resp.NeedOnboarding = false
	}
	return resp, nil
}

func (s *Service) CreateFamily(ctx context.Context, userID uint64, input CreateFamilyInput) (*FamilyResponse, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperror.BadRequest("家庭名称不能为空", nil)
	}

	existing, err := s.repo.GetMembershipByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.Internal("查询家庭失败", err)
	}
	if existing != nil {
		return nil, apperror.Conflict("你已经加入家庭", nil)
	}

	membership, err := s.repo.CreateFamily(ctx, name, userID)
	if err != nil {
		return nil, apperror.Internal("创建家庭失败", err)
	}
	invite, err := s.repo.EnsureActiveInvite(ctx, membership.Family.ID, userID)
	if err != nil {
		return nil, apperror.Internal("创建邀请码失败", err)
	}
	return s.toFamilyResponse(&membership.Family, &membership.Member, invite), nil
}

func (s *Service) JoinFamily(ctx context.Context, userID uint64, input JoinFamilyInput) (*JoinRequestResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" {
		return nil, apperror.BadRequest("邀请码不能为空", nil)
	}

	joinRequest, err := s.repo.CreateJoinRequestByInvite(ctx, code, userID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAlreadyInFamily):
			return nil, apperror.Conflict("你已经加入家庭", err)
		case errors.Is(err, repository.ErrInviteNotFound):
			return nil, apperror.NotFound("邀请码不存在或已失效", nil)
		default:
			return nil, apperror.Internal("提交加入申请失败", err)
		}
	}
	if joinRequest == nil {
		return nil, apperror.NotFound("邀请码不存在或已失效", nil)
	}

	return s.toJoinRequestResponse(joinRequest), nil
}

func (s *Service) ListJoinRequests(ctx context.Context, userID uint64) ([]JoinRequestResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, true)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return nil, apperror.NotFound("请先创建家庭", nil)
	}

	requests, err := s.repo.ListJoinRequestsByFamily(ctx, scope.Membership.Family.ID, model.JoinRequestStatusPending)
	if err != nil {
		return nil, apperror.Internal("查询加入申请失败", err)
	}

	responseItems := make([]JoinRequestResponse, 0, len(requests))
	for _, item := range requests {
		contextItem := item
		responseItems = append(responseItems, *s.toJoinRequestResponse(&contextItem))
	}
	return responseItems, nil
}

func (s *Service) ReviewJoinRequest(ctx context.Context, userID uint64, input ReviewJoinRequestInput) (*JoinRequestResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, true)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return nil, apperror.NotFound("请先创建家庭", nil)
	}

	joinRequest, err := s.repo.ReviewJoinRequest(ctx, scope.Membership.Family.ID, input.RequestID, userID, input.Approve)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrJoinRequestNotFound):
			return nil, apperror.NotFound("申请不存在", nil)
		case errors.Is(err, repository.ErrJoinRequestHandled):
			return nil, apperror.Conflict("申请已处理", err)
		case errors.Is(err, repository.ErrAlreadyInFamily):
			return nil, apperror.Conflict("申请人已经加入其他家庭", err)
		default:
			return nil, apperror.Internal("处理加入申请失败", err)
		}
	}
	if joinRequest == nil {
		return nil, apperror.NotFound("申请不存在", nil)
	}
	return s.toJoinRequestResponse(joinRequest), nil
}

func (s *Service) GetCurrentFamily(ctx context.Context, userID uint64) (*FamilyResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return nil, nil
	}
	invite, err := s.repo.EnsureActiveInvite(ctx, scope.Membership.Family.ID, userID)
	if err != nil {
		return nil, apperror.Internal("获取邀请码失败", err)
	}
	return s.toFamilyResponse(&scope.Membership.Family, &scope.Membership.Member, invite), nil
}

func (s *Service) GetCurrentBaby(ctx context.Context, userID uint64) (*BabyResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Baby == nil {
		return nil, nil
	}
	return s.toBabyResponse(scope.Baby), nil
}

func (s *Service) CreateBaby(ctx context.Context, userID uint64, input UpsertBabyInput) (*BabyResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, true)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return nil, apperror.NotFound("请先创建或加入家庭", nil)
	}
	birthday, err := s.parseDate(input.Birthday)
	if err != nil {
		return nil, apperror.BadRequest("生日格式不正确", err)
	}
	baby := &model.Baby{
		FamilyID:    scope.Membership.Family.ID,
		Name:        strings.TrimSpace(input.Name),
		Gender:      normalizeGender(input.Gender),
		Birthday:    birthday,
		AvatarURL:   strings.TrimSpace(input.AvatarURL),
		FeedingMode: strings.TrimSpace(input.FeedingMode),
		AllergyNote: strings.TrimSpace(input.AllergyNote),
		IsActive:    true,
		CreatedBy:   userID,
	}
	if baby.Name == "" {
		return nil, apperror.BadRequest("宝宝姓名不能为空", nil)
	}

	created, err := s.repo.CreateOrReplaceCurrentBaby(ctx, baby)
	if err != nil {
		return nil, apperror.Internal("保存宝宝信息失败", err)
	}
	return s.toBabyResponse(created), nil
}

func (s *Service) UpdateBaby(ctx context.Context, userID, babyID uint64, input UpsertBabyInput) (*BabyResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, true)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return nil, apperror.NotFound("请先创建或加入家庭", nil)
	}
	if scope.Baby == nil {
		return nil, apperror.NotFound("宝宝档案不存在", nil)
	}
	if scope.Baby.ID != babyID {
		return nil, apperror.NotFound("宝宝档案不存在", nil)
	}

	birthday, err := s.parseDate(input.Birthday)
	if err != nil {
		return nil, apperror.BadRequest("生日格式不正确", err)
	}

	scope.Baby.Name = strings.TrimSpace(input.Name)
	scope.Baby.Gender = normalizeGender(input.Gender)
	scope.Baby.Birthday = birthday
	scope.Baby.AvatarURL = strings.TrimSpace(input.AvatarURL)
	scope.Baby.FeedingMode = strings.TrimSpace(input.FeedingMode)
	scope.Baby.AllergyNote = strings.TrimSpace(input.AllergyNote)

	if scope.Baby.Name == "" {
		return nil, apperror.BadRequest("宝宝姓名不能为空", nil)
	}
	if err := s.repo.UpdateBaby(ctx, scope.Baby); err != nil {
		return nil, apperror.Internal("更新宝宝信息失败", err)
	}
	return s.toBabyResponse(scope.Baby), nil
}

func (s *Service) GetDashboard(ctx context.Context, userID uint64, dayString string) (*DashboardResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Baby == nil {
		return nil, apperror.NotFound("请先创建宝宝档案", nil)
	}

	day, err := s.parseDayOrToday(dayString)
	if err != nil {
		return nil, apperror.BadRequest("日期格式不正确", err)
	}

	records, err := s.repo.ListRecordsByDate(ctx, scope.Membership.Family.ID, scope.Baby.ID, day, "")
	if err != nil {
		return nil, apperror.Internal("查询记录失败", err)
	}
	latestGrowth, err := s.repo.GetLatestGrowthRecord(ctx, scope.Membership.Family.ID, scope.Baby.ID)
	if err != nil {
		return nil, apperror.Internal("查询成长记录失败", err)
	}

	summary := aggregateDashboardSummary(records)
	recent := make([]RecordResponse, 0, len(records))
	for _, record := range records {
		recent = append(recent, s.toRecordResponse(record, scope.Membership.Member, userID))
	}

	resp := &DashboardResponse{
		Date:          day.Format("2006-01-02"),
		Baby:          s.toBabyResponse(scope.Baby),
		Summary:       summary,
		RecentRecords: recent,
	}
	if latestGrowth != nil {
		record := s.toRecordResponse(*latestGrowth, scope.Membership.Member, userID)
		resp.LatestGrowth = &record
	}
	return resp, nil
}

func (s *Service) ListRecords(ctx context.Context, userID uint64, dayString, recordType string) ([]RecordResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Baby == nil {
		return nil, apperror.NotFound("请先创建宝宝档案", nil)
	}

	day, err := s.parseDayOrToday(dayString)
	if err != nil {
		return nil, apperror.BadRequest("日期格式不正确", err)
	}

	if recordType != "" && !isValidRecordType(recordType) {
		return nil, apperror.BadRequest("记录类型不支持", nil)
	}

	records, err := s.repo.ListRecordsByDate(ctx, scope.Membership.Family.ID, scope.Baby.ID, day, recordType)
	if err != nil {
		return nil, apperror.Internal("查询记录失败", err)
	}

	resp := make([]RecordResponse, 0, len(records))
	for _, record := range records {
		resp = append(resp, s.toRecordResponse(record, scope.Membership.Member, userID))
	}
	return resp, nil
}

func (s *Service) CreateRecord(ctx context.Context, userID uint64, input UpsertRecordInput) (*RecordResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Baby == nil {
		return nil, apperror.NotFound("请先创建宝宝档案", nil)
	}

	record, err := s.normalizeRecordPayload(input, scope.Membership.Family.ID, scope.Baby.ID, userID)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.CreateRecord(ctx, record)
	if err != nil {
		return nil, apperror.Internal("创建记录失败", err)
	}

	resp := s.toRecordResponse(*created, scope.Membership.Member, userID)
	return &resp, nil
}

func (s *Service) UpdateRecord(ctx context.Context, userID, recordID uint64, input UpsertRecordInput) (*RecordResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Membership == nil {
		return nil, apperror.NotFound("请先创建或加入家庭", nil)
	}
	record, err := s.repo.GetRecordByID(ctx, recordID)
	if err != nil {
		return nil, apperror.Internal("查询记录失败", err)
	}
	if record == nil || record.FamilyID != scope.Membership.Family.ID {
		return nil, apperror.NotFound("记录不存在", nil)
	}
	if !canManageRecord(scope.Membership.Member, userID, *record) {
		return nil, apperror.Forbidden("没有权限编辑这条记录", nil)
	}

	updated, err := s.normalizeRecordPayload(input, record.FamilyID, record.BabyID, record.CreatedBy)
	if err != nil {
		return nil, err
	}
	updated.ID = record.ID
	updated.CreatedAt = record.CreatedAt
	updated.CreatedBy = record.CreatedBy
	updated.UpdatedBy = userID

	if err := s.repo.UpdateRecord(ctx, updated); err != nil {
		return nil, apperror.Internal("更新记录失败", err)
	}

	resp := s.toRecordResponse(*updated, scope.Membership.Member, userID)
	return &resp, nil
}

func (s *Service) DeleteRecord(ctx context.Context, userID, recordID uint64) error {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return err
	}
	if scope == nil || scope.Membership == nil {
		return apperror.NotFound("请先创建或加入家庭", nil)
	}
	record, err := s.repo.GetRecordByID(ctx, recordID)
	if err != nil {
		return apperror.Internal("查询记录失败", err)
	}
	if record == nil || record.FamilyID != scope.Membership.Family.ID {
		return apperror.NotFound("记录不存在", nil)
	}
	if !canManageRecord(scope.Membership.Member, userID, *record) {
		return apperror.Forbidden("没有权限删除这条记录", nil)
	}
	if err := s.repo.DeleteRecord(ctx, recordID); err != nil {
		return apperror.Internal("删除记录失败", err)
	}
	return nil
}

func (s *Service) GetStats(ctx context.Context, userID uint64, input StatsQueryInput) (*StatsResponse, error) {
	scope, err := s.getFamilyScope(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.Baby == nil {
		return nil, apperror.NotFound("请先创建宝宝档案", nil)
	}

	start, end, days, rangeValue, err := s.resolveStatsWindow(input)
	if err != nil {
		return nil, apperror.BadRequest("统计范围不支持", err)
	}

	records, err := s.repo.ListRecordsInRange(ctx, scope.Membership.Family.ID, scope.Baby.ID, start, end)
	if err != nil {
		return nil, apperror.Internal("查询统计数据失败", err)
	}

	summary, series, growth := aggregateStats(records, start, days, s.location)
	return &StatsResponse{
		Range:       rangeValue,
		StartDate:   start.In(s.location).Format("2006-01-02"),
		EndDate:     end.In(s.location).Format("2006-01-02"),
		Summary:     summary,
		Series:      series,
		GrowthTrend: growth,
	}, nil
}

func (s *Service) getFamilyScope(ctx context.Context, userID uint64, requireOwner bool) (*familyScope, error) {
	membership, err := s.repo.GetMembershipByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.Internal("查询家庭失败", err)
	}
	if membership == nil {
		return nil, nil
	}
	if requireOwner && membership.Member.Role != model.MemberRoleOwner {
		return nil, apperror.Forbidden("只有家庭创建者可以执行此操作", nil)
	}

	baby, err := s.repo.GetCurrentBabyByFamilyID(ctx, membership.Family.ID)
	if err != nil {
		return nil, apperror.Internal("查询宝宝档案失败", err)
	}
	return &familyScope{
		Membership: membership,
		Baby:       baby,
	}, nil
}

func (s *Service) ensureDemoData(ctx context.Context, userID uint64) error {
	existing, err := s.repo.GetMembershipByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	membership, err := s.repo.CreateFamily(ctx, "桃桃成长小队", userID)
	if err != nil {
		return err
	}

	babyBirthday, err := s.parseDate("2025-09-18")
	if err != nil {
		return err
	}

	baby, err := s.repo.CreateOrReplaceCurrentBaby(ctx, &model.Baby{
		FamilyID:    membership.Family.ID,
		Name:        "桃桃",
		Gender:      "girl",
		Birthday:    babyBirthday,
		FeedingMode: "mixed",
		AllergyNote: "夜里容易醒，白天精神很好",
		IsActive:    true,
		CreatedBy:   userID,
	})
	if err != nil {
		return err
	}

	now := time.Now().In(s.location)
	baseDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	for i := 0; i < 10; i += 1 {
		day := baseDay.AddDate(0, 0, -i)
		if err := s.seedDayRecords(ctx, membership.Family.ID, baby.ID, userID, day, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) seedDayRecords(ctx context.Context, familyID, babyID, userID uint64, day time.Time, index int) error {
	feedAmount := float64(150 + (index%3)*15)
	snackAmount := float64(80 + (index%2)*10)
	weight := float64(8.2 + float64(index)*0.03)
	height := float64(71.0 + float64(index)*0.08)
	head := float64(44.2 + float64(index)*0.02)
	sleepDuration := 70 + (index%3)*15

	records := []*model.Record{
		{
			FamilyID:   familyID,
			BabyID:     babyID,
			Type:       model.RecordTypeFeed,
			Subtype:    "formula",
			OccurredAt: atClock(day, 7, 35),
			Amount:     floatPtr(feedAmount),
			Unit:       "ml",
			Note:       "起床第一顿",
			CreatedBy:  userID,
			UpdatedBy:  userID,
		},
		{
			FamilyID:   familyID,
			BabyID:     babyID,
			Type:       model.RecordTypeDiaper,
			Subtype:    "pee",
			OccurredAt: atClock(day, 9, 15),
			CreatedBy:  userID,
			UpdatedBy:  userID,
		},
		{
			FamilyID:   familyID,
			BabyID:     babyID,
			Type:       model.RecordTypeFeed,
			Subtype:    "solid",
			OccurredAt: atClock(day, 11, 20),
			Amount:     floatPtr(snackAmount),
			Unit:       "g",
			Note:       "米粉和南瓜泥",
			CreatedBy:  userID,
			UpdatedBy:  userID,
		},
	}

	sleepStart := atClock(day, 13, 10)
	sleepEnd := sleepStart.Add(time.Duration(sleepDuration) * time.Minute)
	records = append(records, &model.Record{
		FamilyID:    familyID,
		BabyID:      babyID,
		Type:        model.RecordTypeSleep,
		OccurredAt:  sleepStart,
		StartAt:     &sleepStart,
		EndAt:       &sleepEnd,
		DurationMin: intPtr(sleepDuration),
		Note:        "午睡",
		CreatedBy:   userID,
		UpdatedBy:   userID,
	})

	if index%2 == 0 {
		records = append(records, &model.Record{
			FamilyID:   familyID,
			BabyID:     babyID,
			Type:       model.RecordTypeDiaper,
			Subtype:    "mixed",
			OccurredAt: atClock(day, 18, 20),
			CreatedBy:  userID,
			UpdatedBy:  userID,
		})
	}

	if index%3 == 0 {
		records = append(records, &model.Record{
			FamilyID:            familyID,
			BabyID:              babyID,
			Type:                model.RecordTypeGrowth,
			OccurredAt:          atClock(day, 10, 30),
			WeightKg:            floatPtr(weight),
			HeightCm:            floatPtr(height),
			HeadCircumferenceCm: floatPtr(head),
			Note:                "状态很棒，精神很好",
			CreatedBy:           userID,
			UpdatedBy:           userID,
		})
	}

	for _, record := range records {
		if _, err := s.repo.CreateRecord(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) normalizeRecordPayload(input UpsertRecordInput, familyID, babyID, createdBy uint64) (*model.Record, error) {
	recordType := strings.TrimSpace(input.Type)
	if !isValidRecordType(recordType) {
		return nil, apperror.BadRequest("记录类型不支持", nil)
	}

	record := &model.Record{
		FamilyID:  familyID,
		BabyID:    babyID,
		Type:      recordType,
		Subtype:   strings.TrimSpace(input.Subtype),
		Unit:      strings.TrimSpace(input.Unit),
		Note:      strings.TrimSpace(input.Note),
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}

	switch recordType {
	case model.RecordTypeFeed:
		occurredAt, err := s.parseDateTime(input.OccurredAt)
		if err != nil {
			return nil, apperror.BadRequest("喂养时间格式不正确", err)
		}
		record.OccurredAt = occurredAt
		record.Amount = input.Amount
		record.DurationMin = input.DurationMin
		if record.Subtype == "" {
			return nil, apperror.BadRequest("请选择喂养类型", nil)
		}
	case model.RecordTypeSleep:
		startAt, err := s.parseDateTime(input.StartAt)
		if err != nil {
			return nil, apperror.BadRequest("睡眠开始时间格式不正确", err)
		}
		endAt, err := s.parseDateTime(input.EndAt)
		if err != nil {
			return nil, apperror.BadRequest("睡眠结束时间格式不正确", err)
		}
		if !endAt.After(startAt) {
			return nil, apperror.BadRequest("睡眠结束时间必须晚于开始时间", nil)
		}
		duration := int(endAt.Sub(startAt).Minutes())
		record.OccurredAt = startAt
		record.StartAt = &startAt
		record.EndAt = &endAt
		record.DurationMin = &duration
	case model.RecordTypeDiaper:
		occurredAt, err := s.parseDateTime(input.OccurredAt)
		if err != nil {
			return nil, apperror.BadRequest("尿布时间格式不正确", err)
		}
		record.OccurredAt = occurredAt
		if record.Subtype == "" {
			return nil, apperror.BadRequest("请选择尿布类型", nil)
		}
	case model.RecordTypeGrowth:
		occurredAt, err := s.parseDateTime(input.OccurredAt)
		if err != nil {
			return nil, apperror.BadRequest("成长时间格式不正确", err)
		}
		if input.WeightKg == nil && input.HeightCm == nil && input.HeadCircumferenceCm == nil {
			return nil, apperror.BadRequest("成长记录至少填写一项指标", nil)
		}
		record.OccurredAt = occurredAt
		record.WeightKg = input.WeightKg
		record.HeightCm = input.HeightCm
		record.HeadCircumferenceCm = input.HeadCircumferenceCm
	}

	return record, nil
}

func (s *Service) parseDate(value string) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), s.location)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func (s *Service) parseDayOrToday(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now().In(s.location)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location), nil
	}
	return s.parseDate(value)
}

func (s *Service) parseDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty time")
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		var (
			t   time.Time
			err error
		)
		if layout == time.RFC3339 {
			t, err = time.Parse(layout, value)
			if err == nil {
				return t.In(s.location), nil
			}
			continue
		}
		t, err = time.ParseInLocation(layout, value, s.location)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format")
}

func (s *Service) toUserResponse(user *model.User) UserResponse {
	nickname := strings.TrimSpace(user.Nickname)
	if nickname == "" {
		nickname = fmt.Sprintf("用户%d", user.ID)
	}
	return UserResponse{
		ID:        user.ID,
		Nickname:  nickname,
		AvatarURL: user.AvatarURL,
	}
}

func (s *Service) toFamilyResponse(family *model.Family, member *model.FamilyMember, invite *model.FamilyInvite) *FamilyResponse {
	if family == nil || member == nil {
		return nil
	}
	resp := &FamilyResponse{
		ID:   family.ID,
		Name: family.Name,
		Role: member.Role,
	}
	if invite != nil {
		resp.InviteCode = invite.Code
		resp.InviteExpiresAt = invite.ExpiresAt.In(s.location).Format(time.RFC3339)
	}
	return resp
}

func (s *Service) toJoinRequestResponse(joinRequest *repository.JoinRequestContext) *JoinRequestResponse {
	if joinRequest == nil {
		return nil
	}
	resp := &JoinRequestResponse{
		ID:         joinRequest.Request.ID,
		FamilyID:   joinRequest.Family.ID,
		FamilyName: joinRequest.Family.Name,
		Requester:  s.toUserResponse(&joinRequest.User).Nickname,
		Status:     joinRequest.Request.Status,
		InviteCode: joinRequest.Request.InviteCode,
		CreatedAt:  joinRequest.Request.CreatedAt.In(s.location).Format(time.RFC3339),
	}
	if joinRequest.Request.ReviewedAt != nil {
		resp.ReviewedAt = joinRequest.Request.ReviewedAt.In(s.location).Format(time.RFC3339)
	}
	return resp
}

func (s *Service) toBabyResponse(baby *model.Baby) *BabyResponse {
	if baby == nil {
		return nil
	}
	return &BabyResponse{
		ID:          baby.ID,
		Name:        baby.Name,
		Gender:      baby.Gender,
		Birthday:    baby.Birthday.In(s.location).Format("2006-01-02"),
		AvatarURL:   baby.AvatarURL,
		FeedingMode: baby.FeedingMode,
		AllergyNote: baby.AllergyNote,
	}
}

func (s *Service) toRecordResponse(record model.Record, member model.FamilyMember, currentUserID uint64) RecordResponse {
	resp := RecordResponse{
		ID:         record.ID,
		Type:       record.Type,
		Subtype:    record.Subtype,
		OccurredAt: record.OccurredAt.In(s.location).Format(time.RFC3339),
		Unit:       record.Unit,
		Note:       record.Note,
		CreatedBy:  record.CreatedBy,
		CanEdit:    canManageRecord(member, currentUserID, record),
	}
	if record.StartAt != nil {
		resp.StartAt = record.StartAt.In(s.location).Format(time.RFC3339)
	}
	if record.EndAt != nil {
		resp.EndAt = record.EndAt.In(s.location).Format(time.RFC3339)
	}
	resp.Amount = record.Amount
	resp.DurationMin = record.DurationMin
	resp.WeightKg = record.WeightKg
	resp.HeightCm = record.HeightCm
	resp.HeadCircumferenceCm = record.HeadCircumferenceCm
	return resp
}

func aggregateDashboardSummary(records []model.Record) DashboardSummary {
	var summary DashboardSummary
	for _, record := range records {
		switch record.Type {
		case model.RecordTypeFeed:
			summary.FeedCount++
		case model.RecordTypeSleep:
			summary.SleepCount++
			if record.DurationMin != nil {
				summary.SleepDurationMin += *record.DurationMin
			}
		case model.RecordTypeDiaper:
			summary.DiaperCount++
		case model.RecordTypeGrowth:
			summary.GrowthCount++
		}
	}
	return summary
}

func aggregateStats(records []model.Record, start time.Time, days int, loc *time.Location) (DashboardSummary, []DailyStat, []GrowthPoint) {
	summary := aggregateDashboardSummary(records)
	series := make([]DailyStat, 0, days)
	daily := map[string]*DailyStat{}

	for i := range days {
		day := start.AddDate(0, 0, i).In(loc).Format("2006-01-02")
		stat := &DailyStat{Date: day}
		daily[day] = stat
		series = append(series, *stat)
	}

	growth := make([]GrowthPoint, 0)
	for _, record := range records {
		day := record.OccurredAt.In(loc).Format("2006-01-02")
		stat, ok := daily[day]
		if ok {
			switch record.Type {
			case model.RecordTypeFeed:
				stat.FeedCount++
			case model.RecordTypeSleep:
				if record.DurationMin != nil {
					stat.SleepDurationMin += *record.DurationMin
				}
			case model.RecordTypeDiaper:
				stat.DiaperCount++
			}
		}
		if record.Type == model.RecordTypeGrowth {
			growth = append(growth, GrowthPoint{
				Date:                record.OccurredAt.In(loc).Format("2006-01-02"),
				WeightKg:            record.WeightKg,
				HeightCm:            record.HeightCm,
				HeadCircumferenceCm: record.HeadCircumferenceCm,
			})
		}
	}

	for idx := range series {
		if stat, ok := daily[series[idx].Date]; ok {
			series[idx] = *stat
		}
	}
	return summary, series, growth
}

func parseRange(rangeValue string) (int, error) {
	switch rangeValue {
	case "7d", "":
		return 7, nil
	case "30d":
		return 30, nil
	default:
		return 0, fmt.Errorf("invalid range")
	}
}

func (s *Service) resolveStatsWindow(input StatsQueryInput) (time.Time, time.Time, int, string, error) {
	startText := strings.TrimSpace(input.StartDate)
	endText := strings.TrimSpace(input.EndDate)

	if startText != "" || endText != "" {
		if startText == "" || endText == "" {
			return time.Time{}, time.Time{}, 0, "", fmt.Errorf("startDate and endDate must both be set")
		}
		startDay, err := s.parseDate(startText)
		if err != nil {
			return time.Time{}, time.Time{}, 0, "", err
		}
		endDay, err := s.parseDate(endText)
		if err != nil {
			return time.Time{}, time.Time{}, 0, "", err
		}
		if endDay.Before(startDay) {
			return time.Time{}, time.Time{}, 0, "", fmt.Errorf("endDate before startDate")
		}
		start := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, s.location)
		end := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), s.location)
		days := int(end.Sub(start)/(24*time.Hour)) + 1
		return start, end, days, "custom", nil
	}

	days, err := parseRange(input.Range)
	if err != nil {
		return time.Time{}, time.Time{}, 0, "", err
	}

	now := time.Now().In(s.location)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location).AddDate(0, 0, -(days - 1))
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), s.location)
	rangeValue := strings.TrimSpace(input.Range)
	if rangeValue == "" {
		rangeValue = "7d"
	}
	return start, end, days, rangeValue, nil
}

func isValidRecordType(recordType string) bool {
	switch recordType {
	case model.RecordTypeFeed, model.RecordTypeSleep, model.RecordTypeDiaper, model.RecordTypeGrowth:
		return true
	default:
		return false
	}
}

func normalizeGender(value string) string {
	switch strings.TrimSpace(value) {
	case "boy", "girl":
		return strings.TrimSpace(value)
	default:
		return "unknown"
	}
}

func canManageRecord(member model.FamilyMember, userID uint64, record model.Record) bool {
	return member.Role == model.MemberRoleOwner || record.CreatedBy == userID
}

func atClock(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}
