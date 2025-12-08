package models

import (
	"time"

	"gorm.io/gorm"
)

type CalculationStatus string

const (
	CalculationStatusDraft     CalculationStatus = "черновик"
	CalculationStatusDeleted   CalculationStatus = "удалён"
	CalculationStatusFormed    CalculationStatus = "сформирован"
	CalculationStatusCompleted CalculationStatus = "завершён"
	CalculationStatusRejected  CalculationStatus = "отклонён"
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

type GlomerularCalculation struct {
	ID              uint               `gorm:"primaryKey"`
	Status          CalculationStatus  `gorm:"type:varchar(20);not null;default:'черновик'"`
	CreatedAt       time.Time          `gorm:"not null"`
	CreatorID       uint               `gorm:"not null"`
	DoctorName      *string            `gorm:"type:varchar(255);column:doctor_name"`
	AverageAge      *float64           `gorm:"type:decimal(10,2);column:average_age"`
	FormedAt        *time.Time
	CompletedAt     *time.Time
	ModeratorID     *uint
	CalculatedGFR   *float64                   `gorm:"type:decimal(10,2)"`
	Categories      []PatientCategory           `gorm:"many2many:glomerular_calculation_categories;"`
	CalculationCategories []GlomerularCalculationCategory `gorm:"foreignKey:CalculationID;references:ID"`
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (GlomerularCalculation) TableName() string {
	return "glomerular_calculations"
}

type GlomerularCalculationCategory struct {
	CalculationID   uint      `gorm:"primaryKey;column:calculation_id"`
	CategoryID      uint      `gorm:"primaryKey;column:category_id"`
	Quantity        *int      `gorm:"type:integer;column:quantity"`
	OrderNum        *int      `gorm:"type:integer;column:order_num"`
	IsMain          *bool     `gorm:"type:boolean;column:is_main"`
	CreatinineLevel *float64  `gorm:"type:decimal(10,2);column:creatinine_level"`
	CalculatedGFR   *float64  `gorm:"type:decimal(10,2);column:calculated_gfr"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (GlomerularCalculationCategory) TableName() string {
	return "glomerular_calculation_categories"
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
