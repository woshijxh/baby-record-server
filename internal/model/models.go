package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	MemberRoleOwner  = "owner"
	MemberRoleMember = "member"

	JoinRequestStatusPending  = "pending"
	JoinRequestStatusApproved = "approved"
	JoinRequestStatusRejected = "rejected"

	RecordTypeFeed   = "feed"
	RecordTypeSleep  = "sleep"
	RecordTypeDiaper = "diaper"
	RecordTypeGrowth = "growth"
)

type User struct {
	ID        uint64    `gorm:"primaryKey"`
	OpenID    string    `gorm:"size:128;not null;uniqueIndex"`
	Nickname  string    `gorm:"size:64;not null;default:''"`
	AvatarURL string    `gorm:"size:255;not null;default:''"`
	Status    int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (User) TableName() string { return "users" }

type Family struct {
	ID          uint64    `gorm:"primaryKey"`
	Name        string    `gorm:"size:64;not null"`
	OwnerUserID uint64    `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (Family) TableName() string { return "families" }

type FamilyMember struct {
	ID        uint64    `gorm:"primaryKey"`
	FamilyID  uint64    `gorm:"not null;index:idx_family_user,unique"`
	UserID    uint64    `gorm:"not null;index:idx_family_user,unique;index"`
	Role      string    `gorm:"size:16;not null"`
	Status    string    `gorm:"size:16;not null;default:'active';index"`
	JoinedAt  time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (FamilyMember) TableName() string { return "family_members" }

type FamilyInvite struct {
	ID        uint64     `gorm:"primaryKey"`
	FamilyID  uint64     `gorm:"not null;index"`
	Code      string     `gorm:"size:6;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	UsedBy    *uint64    `gorm:"index"`
	UsedAt    *time.Time `gorm:"index"`
	CreatedBy uint64     `gorm:"not null"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt time.Time  `gorm:"not null"`
}

func (FamilyInvite) TableName() string { return "family_invites" }

type FamilyJoinRequest struct {
	ID         uint64     `gorm:"primaryKey"`
	FamilyID   uint64     `gorm:"not null;index"`
	UserID     uint64     `gorm:"not null;index"`
	InviteCode string     `gorm:"size:6;not null;index"`
	Status     string     `gorm:"size:16;not null;default:'pending';index"`
	ReviewedBy *uint64    `gorm:"index"`
	ReviewedAt *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"not null"`
	UpdatedAt  time.Time  `gorm:"not null"`
}

func (FamilyJoinRequest) TableName() string { return "family_join_requests" }

type Baby struct {
	ID          uint64    `gorm:"primaryKey"`
	FamilyID    uint64    `gorm:"not null;index"`
	Name        string    `gorm:"size:64;not null"`
	Gender      string    `gorm:"size:16;not null;default:'unknown'"`
	Birthday    time.Time `gorm:"not null"`
	AvatarURL   string    `gorm:"size:255;not null;default:''"`
	FeedingMode string    `gorm:"size:32;not null;default:''"`
	AllergyNote string    `gorm:"size:255;not null;default:''"`
	IsActive    bool      `gorm:"not null;default:true;index"`
	CreatedBy   uint64    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (Baby) TableName() string { return "babies" }

type Record struct {
	ID                  uint64         `gorm:"primaryKey"`
	FamilyID            uint64         `gorm:"not null;index"`
	BabyID              uint64         `gorm:"not null;index"`
	Type                string         `gorm:"size:16;not null;index"`
	Subtype             string         `gorm:"size:32;not null;default:''"`
	OccurredAt          time.Time      `gorm:"not null;index"`
	StartAt             *time.Time     `gorm:"index"`
	EndAt               *time.Time     `gorm:"index"`
	Amount              *float64       `gorm:"type:decimal(10,2)"`
	Unit                string         `gorm:"size:16;not null;default:''"`
	DurationMin         *int           `gorm:"type:int"`
	WeightKg            *float64       `gorm:"type:decimal(6,2)"`
	HeightCm            *float64       `gorm:"type:decimal(6,2)"`
	HeadCircumferenceCm *float64       `gorm:"type:decimal(6,2)"`
	Note                string         `gorm:"size:255;not null;default:''"`
	CreatedBy           uint64         `gorm:"not null;index"`
	UpdatedBy           uint64         `gorm:"not null"`
	CreatedAt           time.Time      `gorm:"not null"`
	UpdatedAt           time.Time      `gorm:"not null"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

func (Record) TableName() string { return "records" }
