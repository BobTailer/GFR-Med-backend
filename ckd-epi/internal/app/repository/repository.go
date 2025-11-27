package repository

import (
	"ckd-epi/internal/app/database"
	"ckd-epi/internal/app/models"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() (*Repository, error) {
	return &Repository{db: database.DB}, nil
}

func (r *Repository) GetPatientCategories(searchQuery string) ([]models.PatientCategory, error) {
	var categories []models.PatientCategory
	query := r.db.Where("is_deleted = ?", false)

	if searchQuery != "" {
		searchPattern := "%" + strings.ToLower(searchQuery) + "%"
		query = query.Where("LOWER(name) LIKE ?", searchPattern)
	}

	err := query.Order("id ASC").Find(&categories).Error
	return categories, err
}

func (r *Repository) GetPatientCategoryByID(id uint) (*models.PatientCategory, error) {
	var category models.PatientCategory
	err := r.db.Where("id = ? AND is_deleted = ?", id, false).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *Repository) GetOrCreateDraftGFROrder(userID uint) (*models.GFROrder, error) {
	var gfrOrder models.GFROrder

	err := r.db.Where("creator_id = ? AND status = ? AND deleted_at IS NULL", userID, models.OrderStatusDraft).First(&gfrOrder).Error

	if err == gorm.ErrRecordNotFound {
		gfrOrder = models.GFROrder{
			Status:    models.OrderStatusDraft,
			CreatedAt: time.Now(),
			CreatorID: userID,
		}
		if err := r.db.Create(&gfrOrder).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &gfrOrder, nil
}

func (r *Repository) AddCategoryToGFROrder(orderID, categoryID uint) error {
	var existing models.GFROrderCategory
	err := r.db.Where("order_id = ? AND category_id = ?", orderID, categoryID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		orderCategory := models.GFROrderCategory{
			OrderID:         orderID,
			CategoryID:      categoryID,
			Quantity:        nil,
			OrderNum:        nil,
			IsMain:          nil,
			CreatinineLevel: nil,
		}
		return r.db.Create(&orderCategory).Error
	}

	return nil
}

func (r *Repository) UpdateGFROrderCategoryCreatinine(orderID, categoryID uint, creatinineLevel float64) error {
	var orderCategory models.GFROrderCategory
	err := r.db.Where("order_id = ? AND category_id = ?", orderID, categoryID).First(&orderCategory).Error
	if err != nil {
		return err
	}
	
	orderCategory.CreatinineLevel = &creatinineLevel
	
	return r.db.Save(&orderCategory).Error
}

func (r *Repository) GetGFROrderByID(orderID uint) (*models.GFROrder, error) {
	var gfrOrder models.GFROrder
	err := r.db.Where("id = ? AND deleted_at IS NULL", orderID).First(&gfrOrder).Error
	if err != nil {
		return nil, err
	}
	
	err = r.db.Select("order_id", "category_id", "quantity", "order_num", "is_main", "creatinine_level", "created_at", "updated_at").
		Where("order_id = ?", orderID).
		Find(&gfrOrder.OrderCategories).Error
	if err != nil {
		return nil, err
	}
	
	if len(gfrOrder.OrderCategories) > 0 {
		var categoryIDs []uint
		for _, oc := range gfrOrder.OrderCategories {
			categoryIDs = append(categoryIDs, oc.CategoryID)
		}
		err = r.db.Where("id IN ? AND deleted_at IS NULL", categoryIDs).Find(&gfrOrder.Categories).Error
		if err != nil {
			return nil, err
		}
	}
	
	return &gfrOrder, nil
}

func (r *Repository) DeleteGFROrder(orderID uint) error {
	result := r.db.Exec(`
		UPDATE gfr_orders 
		SET status = ?, updated_at = ? 
		WHERE id = ? AND deleted_at IS NULL
	`, models.OrderStatusDeleted, time.Now(), orderID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("order not found or already deleted")
	}

	return nil
}

func (r *Repository) CalculateGFR(gfrOrder *models.GFROrder) float64 {
	if len(gfrOrder.OrderCategories) == 0 {
		return 0
	}

	var totalGFR float64
	var count int

	for _, orderCategory := range gfrOrder.OrderCategories {
		if orderCategory.CreatinineLevel == nil {
			continue
		}

		var category models.PatientCategory
		for _, c := range gfrOrder.Categories {
			if c.ID == orderCategory.CategoryID {
				category = c
				break
			}
		}

		if category.ID == 0 {
			continue
		}

		creatinine := *orderCategory.CreatinineLevel

		var gfr float64
		if category.Gender == "М" {
			gfr = 175 * math.Pow(creatinine/88.4, -1.154) * math.Pow(float64(category.Age), -0.203) * 0.742
		} else {
			gfr = 175 * math.Pow(creatinine/88.4, -1.154) * math.Pow(float64(category.Age), -0.203) * 0.742 * 1.018
		}

		if gfr < 0 {
			gfr = 0
		}
		if gfr > 200 {
			gfr = 200
		}

		totalGFR += gfr
		count++
	}

	if count == 0 {
		return 0
	}

	return totalGFR / float64(count)
}

func (r *Repository) CompleteGFROrder(orderID, moderatorID uint) error {
	gfrOrder, err := r.GetGFROrderByID(orderID)
	if err != nil {
		return err
	}

	gfr := r.CalculateGFR(gfrOrder)
	now := time.Now()

	return r.db.Model(&models.GFROrder{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":         models.OrderStatusCompleted,
			"completed_at":   &now,
			"moderator_id":   moderatorID,
			"calculated_gfr": gfr,
		}).Error
}

func (r *Repository) GetDraftGFROrder(userID uint) (*models.GFROrder, error) {
	var gfrOrder models.GFROrder
	err := r.db.Where("creator_id = ? AND status = ? AND deleted_at IS NULL", userID, models.OrderStatusDraft).First(&gfrOrder).Error
	if err != nil {
		return nil, err
	}
	return &gfrOrder, nil
}

func (r *Repository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetDefaultUser() (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", "admin").First(&user).Error
	if err != nil {
		user = models.User{
			Username:    "admin",
			Password:    "admin",
			IsModerator: true,
		}
		if err := r.db.Create(&user).Error; err != nil {
			return nil, err
		}
	}
	return &user, nil
}

