package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"baby-record-server/internal/model"
	"gorm.io/gorm"
)

var (
	ErrAlreadyInFamily   = errors.New("user already belongs to a family")
	ErrInviteNotFound    = errors.New("invite code not found or expired")
	ErrJoinRequestNotFound = errors.New("join request not found")
	ErrJoinRequestHandled  = errors.New("join request already handled")
)

type MembershipContext struct {
	Family model.Family
	Member model.FamilyMember
}

type JoinRequestContext struct {
	Request model.FamilyJoinRequest
	Family  model.Family
	User    model.User
}

type Repository interface {
	Migrate(ctx context.Context) error
	UpsertUserByOpenID(ctx context.Context, openID, nickname, avatarURL string) (*model.User, error)
	GetUserByID(ctx context.Context, userID uint64) (*model.User, error)
	GetMembershipByUserID(ctx context.Context, userID uint64) (*MembershipContext, error)
	CreateFamily(ctx context.Context, name string, ownerUserID uint64) (*MembershipContext, error)
	EnsureActiveInvite(ctx context.Context, familyID, createdBy uint64) (*model.FamilyInvite, error)
	CreateJoinRequestByInvite(ctx context.Context, code string, userID uint64) (*JoinRequestContext, error)
	ListJoinRequestsByFamily(ctx context.Context, familyID uint64, status string) ([]JoinRequestContext, error)
	ReviewJoinRequest(ctx context.Context, familyID, requestID, reviewerID uint64, approve bool) (*JoinRequestContext, error)
	GetCurrentBabyByFamilyID(ctx context.Context, familyID uint64) (*model.Baby, error)
	CreateOrReplaceCurrentBaby(ctx context.Context, baby *model.Baby) (*model.Baby, error)
	UpdateBaby(ctx context.Context, baby *model.Baby) error
	CreateRecord(ctx context.Context, record *model.Record) (*model.Record, error)
	GetRecordByID(ctx context.Context, recordID uint64) (*model.Record, error)
	UpdateRecord(ctx context.Context, record *model.Record) error
	DeleteRecord(ctx context.Context, recordID uint64) error
	ListRecordsByDate(ctx context.Context, familyID, babyID uint64, day time.Time, recordType string) ([]model.Record, error)
	ListRecordsInRange(ctx context.Context, familyID, babyID uint64, from, to time.Time) ([]model.Record, error)
	GetLatestGrowthRecord(ctx context.Context, familyID, babyID uint64) (*model.Record, error)
}

type GormRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Migrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(
		&model.User{},
		&model.Family{},
		&model.FamilyMember{},
		&model.FamilyInvite{},
		&model.FamilyJoinRequest{},
		&model.Baby{},
		&model.Record{},
	)
}

func (r *GormRepository) UpsertUserByOpenID(ctx context.Context, openID, nickname, avatarURL string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("open_id = ?", openID).First(&user).Error
	switch {
	case err == nil:
		updates := map[string]interface{}{}
		if nickname != "" {
			updates["nickname"] = nickname
		}
		if avatarURL != "" {
			updates["avatar_url"] = avatarURL
		}
		if len(updates) > 0 {
			if err := r.db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
				return nil, err
			}
			if err := r.db.WithContext(ctx).First(&user, user.ID).Error; err != nil {
				return nil, err
			}
		}
		return &user, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		user = model.User{
			OpenID:    openID,
			Nickname:  nickname,
			AvatarURL: avatarURL,
			Status:    1,
		}
		if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	default:
		return nil, err
	}
}

func (r *GormRepository) GetUserByID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) GetMembershipByUserID(ctx context.Context, userID uint64) (*MembershipContext, error) {
	type membershipRow struct {
		Member model.FamilyMember
		Family model.Family
	}

	var member model.FamilyMember
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "active").
		Order("joined_at asc").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var family model.Family
	if err := r.db.WithContext(ctx).First(&family, member.FamilyID).Error; err != nil {
		return nil, err
	}
	return &MembershipContext{Family: family, Member: member}, nil
}

func (r *GormRepository) CreateFamily(ctx context.Context, name string, ownerUserID uint64) (*MembershipContext, error) {
	var result MembershipContext
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		family := model.Family{
			Name:        name,
			OwnerUserID: ownerUserID,
		}
		if err := tx.Create(&family).Error; err != nil {
			return err
		}

		member := model.FamilyMember{
			FamilyID: family.ID,
			UserID:   ownerUserID,
			Role:     model.MemberRoleOwner,
			Status:   "active",
			JoinedAt: time.Now(),
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		result = MembershipContext{Family: family, Member: member}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *GormRepository) EnsureActiveInvite(ctx context.Context, familyID, createdBy uint64) (*model.FamilyInvite, error) {
	now := time.Now()
	var invite model.FamilyInvite
	err := r.db.WithContext(ctx).
		Where("family_id = ? AND expires_at > ?", familyID, now).
		Order("id desc").
		First(&invite).Error
	if err == nil {
		return &invite, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	for range 8 {
		code, err := generateInviteCode()
		if err != nil {
			return nil, err
		}
		invite = model.FamilyInvite{
			FamilyID:  familyID,
			Code:      code,
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedBy: createdBy,
		}
		if err := r.db.WithContext(ctx).Create(&invite).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				continue
			}
			return nil, err
		}
		return &invite, nil
	}
	return nil, fmt.Errorf("failed to generate invite code")
}

func (r *GormRepository) CreateJoinRequestByInvite(ctx context.Context, code string, userID uint64) (*JoinRequestContext, error) {
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))
	var result JoinRequestContext
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.FamilyMember
		if err := tx.Where("user_id = ? AND status = ?", userID, "active").First(&existing).Error; err == nil {
			return ErrAlreadyInFamily
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var invite model.FamilyInvite
		if err := tx.Where("code = ? AND expires_at > ?", normalizedCode, time.Now()).First(&invite).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInviteNotFound
			}
			return err
		}

		var request model.FamilyJoinRequest
		err := tx.Where("family_id = ? AND user_id = ? AND status = ?", invite.FamilyID, userID, model.JoinRequestStatusPending).
			Order("id desc").
			First(&request).Error
		switch {
		case err == nil:
		case errors.Is(err, gorm.ErrRecordNotFound):
			request = model.FamilyJoinRequest{
				FamilyID:   invite.FamilyID,
				UserID:     userID,
				InviteCode: normalizedCode,
				Status:     model.JoinRequestStatusPending,
			}
			if err := tx.Create(&request).Error; err != nil {
				return err
			}
		default:
			return err
		}

		var family model.Family
		if err := tx.First(&family, invite.FamilyID).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}

		result = JoinRequestContext{Request: request, Family: family, User: user}
		return nil
	})
	switch {
	case errors.Is(err, ErrInviteNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &result, nil
	}
}

func (r *GormRepository) ListJoinRequestsByFamily(ctx context.Context, familyID uint64, status string) ([]JoinRequestContext, error) {
	query := r.db.WithContext(ctx).Where("family_id = ?", familyID)
	if strings.TrimSpace(status) != "" {
		query = query.Where("status = ?", status)
	}

	var requests []model.FamilyJoinRequest
	if err := query.Order("created_at desc").Find(&requests).Error; err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return []JoinRequestContext{}, nil
	}

	var family model.Family
	if err := r.db.WithContext(ctx).First(&family, familyID).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uint64, 0, len(requests))
	seen := make(map[uint64]struct{}, len(requests))
	for _, request := range requests {
		if _, ok := seen[request.UserID]; ok {
			continue
		}
		seen[request.UserID] = struct{}{}
		userIDs = append(userIDs, request.UserID)
	}

	var users []model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[uint64]model.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	result := make([]JoinRequestContext, 0, len(requests))
	for _, request := range requests {
		result = append(result, JoinRequestContext{
			Request: request,
			Family:  family,
			User:    userMap[request.UserID],
		})
	}
	return result, nil
}

func (r *GormRepository) ReviewJoinRequest(ctx context.Context, familyID, requestID, reviewerID uint64, approve bool) (*JoinRequestContext, error) {
	var result JoinRequestContext
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var request model.FamilyJoinRequest
		if err := tx.Where("id = ? AND family_id = ?", requestID, familyID).First(&request).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrJoinRequestNotFound
			}
			return err
		}
		if request.Status != model.JoinRequestStatusPending {
			return ErrJoinRequestHandled
		}

		var user model.User
		if err := tx.First(&user, request.UserID).Error; err != nil {
			return err
		}
		var family model.Family
		if err := tx.First(&family, familyID).Error; err != nil {
			return err
		}

		status := model.JoinRequestStatusRejected
		if approve {
			var existing model.FamilyMember
			if err := tx.Where("user_id = ? AND status = ?", request.UserID, "active").First(&existing).Error; err == nil {
				return ErrAlreadyInFamily
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			member := model.FamilyMember{
				FamilyID: familyID,
				UserID:   request.UserID,
				Role:     model.MemberRoleMember,
				Status:   "active",
				JoinedAt: time.Now(),
			}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
			status = model.JoinRequestStatusApproved
		}

		reviewedAt := time.Now()
		if err := tx.Model(&request).Updates(map[string]interface{}{
			"status":      status,
			"reviewed_by": reviewerID,
			"reviewed_at": reviewedAt,
		}).Error; err != nil {
			return err
		}
		request.Status = status
		request.ReviewedBy = &reviewerID
		request.ReviewedAt = &reviewedAt
		result = JoinRequestContext{
			Request: request,
			Family:  family,
			User:    user,
		}
		return nil
	})
	switch {
	case err == nil:
		return &result, nil
	case errors.Is(err, ErrJoinRequestNotFound):
		return nil, nil
	default:
		return nil, err
	}
}

func (r *GormRepository) GetCurrentBabyByFamilyID(ctx context.Context, familyID uint64) (*model.Baby, error) {
	var baby model.Baby
	err := r.db.WithContext(ctx).
		Where("family_id = ? AND is_active = ?", familyID, true).
		Order("id desc").
		First(&baby).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &baby, nil
}

func (r *GormRepository) CreateOrReplaceCurrentBaby(ctx context.Context, baby *model.Baby) (*model.Baby, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Baby{}).
			Where("family_id = ?", baby.FamilyID).
			Update("is_active", false).Error; err != nil {
			return err
		}
		if err := tx.Create(baby).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return baby, nil
}

func (r *GormRepository) UpdateBaby(ctx context.Context, baby *model.Baby) error {
	return r.db.WithContext(ctx).Save(baby).Error
}

func (r *GormRepository) CreateRecord(ctx context.Context, record *model.Record) (*model.Record, error) {
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (r *GormRepository) GetRecordByID(ctx context.Context, recordID uint64) (*model.Record, error) {
	var record model.Record
	err := r.db.WithContext(ctx).First(&record, recordID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *GormRepository) UpdateRecord(ctx context.Context, record *model.Record) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *GormRepository) DeleteRecord(ctx context.Context, recordID uint64) error {
	return r.db.WithContext(ctx).Delete(&model.Record{}, recordID).Error
}

func (r *GormRepository) ListRecordsByDate(ctx context.Context, familyID, babyID uint64, day time.Time, recordType string) ([]model.Record, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	query := r.db.WithContext(ctx).
		Where("family_id = ? AND baby_id = ? AND occurred_at >= ? AND occurred_at < ?", familyID, babyID, start, end)
	if recordType != "" {
		query = query.Where("type = ?", recordType)
	}

	var records []model.Record
	if err := query.Order("occurred_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *GormRepository) ListRecordsInRange(ctx context.Context, familyID, babyID uint64, from, to time.Time) ([]model.Record, error) {
	var records []model.Record
	if err := r.db.WithContext(ctx).
		Where("family_id = ? AND baby_id = ? AND occurred_at >= ? AND occurred_at <= ?", familyID, babyID, from, to).
		Order("occurred_at asc").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *GormRepository) GetLatestGrowthRecord(ctx context.Context, familyID, babyID uint64) (*model.Record, error) {
	var record model.Record
	err := r.db.WithContext(ctx).
		Where("family_id = ? AND baby_id = ? AND type = ?", familyID, babyID, model.RecordTypeGrowth).
		Order("occurred_at desc").
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func generateInviteCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	for range 6 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(charset[n.Int64()])
	}
	return builder.String(), nil
}
