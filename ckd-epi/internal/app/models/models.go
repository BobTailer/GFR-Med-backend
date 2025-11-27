package models

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusDraft     OrderStatus = "черновик"
	OrderStatusDeleted   OrderStatus = "удалён"
	OrderStatusFormed    OrderStatus = "сформирован"
	OrderStatusCompleted OrderStatus = "завершён"
	OrderStatusRejected  OrderStatus = "отклонён"
)

type PatientCategory struct {
	ID          uint    `gorm:"primaryKey"`
	Name        string  `gorm:"type:varchar(255);not null"`
	Description string  `gorm:"type:text"`
	IsDeleted   bool    `gorm:"default:false;not null"`
	ImageURL    *string `gorm:"type:varchar(500)"`
	ImageKey    *string `gorm:"type:varchar(255)"`
	Gender      string  `gorm:"type:varchar(10);not null"`
	Age         int     `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (PatientCategory) TableName() string {
	return "patient_categories"
}

type GFROrder struct {
	ID              uint        `gorm:"primaryKey"`
	Status          OrderStatus `gorm:"type:varchar(20);not null;default:'черновик'"`
	CreatedAt       time.Time   `gorm:"not null"`
	CreatorID       uint        `gorm:"not null"`
	FormedAt        *time.Time
	CompletedAt     *time.Time
	ModeratorID     *uint
	CalculatedGFR   *float64           `gorm:"type:decimal(10,2)"`
	Categories      []PatientCategory  `gorm:"many2many:gfr_order_categories;"`
	OrderCategories []GFROrderCategory `gorm:"foreignKey:OrderID;references:ID"`
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (GFROrder) TableName() string {
	return "gfr_orders"
}

type GFROrderCategory struct {
	OrderID         uint      `gorm:"primaryKey;column:order_id"`
	CategoryID      uint      `gorm:"primaryKey;column:category_id"`
	Quantity        *int      `gorm:"type:integer;column:quantity"`
	OrderNum        *int      `gorm:"type:integer;column:order_num"`
	IsMain          *bool     `gorm:"type:boolean;column:is_main"`
	CreatinineLevel *float64  `gorm:"type:decimal(10,2);column:creatinine_level"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (GFROrderCategory) TableName() string {
	return "gfr_order_categories"
}

type User struct {
	ID          uint   `gorm:"primaryKey"`
	Username    string `gorm:"type:varchar(100);unique;not null"`
	Password    string `gorm:"type:varchar(255);not null"`
	IsModerator bool   `gorm:"default:false;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
