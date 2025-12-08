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

func (r *Repository) GetOrCreateDraftGlomerularCalculation(userID uint) (*models.GlomerularCalculation, error) {
	var calculation models.GlomerularCalculation

	err := r.db.Where("creator_id = ? AND status = ? AND deleted_at IS NULL", userID, models.CalculationStatusDraft).First(&calculation).Error

	if err == gorm.ErrRecordNotFound {
		calculation = models.GlomerularCalculation{
			Status:    models.CalculationStatusDraft,
			CreatedAt: time.Now(),
			CreatorID: userID,
		}
		if err := r.db.Create(&calculation).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &calculation, nil
}

func (r *Repository) CalculateAverageAge(calculationID uint) (float64, error) {
	calculation, err := r.GetGlomerularCalculationByID(calculationID)
	if err != nil {
		return 0, err
	}

	if len(calculation.Categories) == 0 {
		return 0, nil
	}

	var totalAge int
	for _, cat := range calculation.Categories {
		totalAge += cat.Age
	}

	averageAge := float64(totalAge) / float64(len(calculation.Categories))
	return averageAge, nil
}

func (r *Repository) AddCategoryToGlomerularCalculation(calculationID, categoryID uint) error {
	var existing models.GlomerularCalculationCategory
	err := r.db.Where("calculation_id = ? AND category_id = ?", calculationID, categoryID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		calculationCategory := models.GlomerularCalculationCategory{
			CalculationID:   calculationID,
			CategoryID:      categoryID,
			Quantity:        nil,
			OrderNum:        nil,
			IsMain:          nil,
			CreatinineLevel: nil,
			CalculatedGFR:   nil,
		}
		if err := r.db.Create(&calculationCategory).Error; err != nil {
			return err
		}

		var category models.PatientCategory
		if err := r.db.Where("id = ?", categoryID).First(&category).Error; err == nil {
			var categoryIDs []uint
			var existingCategories []models.GlomerularCalculationCategory
			r.db.Where("calculation_id = ?", calculationID).Find(&existingCategories)
			
			for _, cc := range existingCategories {
				categoryIDs = append(categoryIDs, cc.CategoryID)
			}
			
			var categories []models.PatientCategory
			if len(categoryIDs) > 0 {
				r.db.Where("id IN ? AND deleted_at IS NULL", categoryIDs).Find(&categories)
			}
			
			var totalAge int
			for _, cat := range categories {
				totalAge += cat.Age
			}
			
			if len(categories) > 0 {
				averageAge := float64(totalAge) / float64(len(categories))
				return r.db.Model(&models.GlomerularCalculation{}).
					Where("id = ?", calculationID).
					Update("average_age", averageAge).Error
			}
		}
		return nil
	}

	return nil
}

func (r *Repository) UpdateGlomerularCalculationCategoryCreatinine(calculationID, categoryID uint, creatinineLevel float64) error {
	var calculationCategory models.GlomerularCalculationCategory
	err := r.db.Where("calculation_id = ? AND category_id = ?", calculationID, categoryID).First(&calculationCategory).Error
	if err != nil {
		return err
	}
	
	calculationCategory.CreatinineLevel = &creatinineLevel
	
	return r.db.Save(&calculationCategory).Error
}

func (r *Repository) GetGlomerularCalculationByID(calculationID uint) (*models.GlomerularCalculation, error) {
	var calculation models.GlomerularCalculation
	err := r.db.Where("id = ? AND deleted_at IS NULL", calculationID).First(&calculation).Error
	if err != nil {
		return nil, err
	}
	
	err = r.db.Select("calculation_id", "category_id", "quantity", "order_num", "is_main", "creatinine_level", "calculated_gfr", "created_at", "updated_at").
		Where("calculation_id = ?", calculationID).
		Find(&calculation.CalculationCategories).Error
	if err != nil {
		return nil, err
	}
	
	if len(calculation.CalculationCategories) > 0 {
		var categoryIDs []uint
		for _, cc := range calculation.CalculationCategories {
			categoryIDs = append(categoryIDs, cc.CategoryID)
		}
		err = r.db.Where("id IN ? AND deleted_at IS NULL", categoryIDs).Find(&calculation.Categories).Error
		if err != nil {
			return nil, err
		}
	}
	
	return &calculation, nil
}

func (r *Repository) DeleteGlomerularCalculation(calculationID uint) error {
	result := r.db.Exec(`
		UPDATE glomerular_calculations 
		SET status = ?, updated_at = ? 
		WHERE id = ? AND deleted_at IS NULL
	`, models.CalculationStatusDeleted, time.Now(), calculationID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("calculation not found or already deleted")
	}

	return nil
}

func (r *Repository) CalculateGFR(calculation *models.GlomerularCalculation) float64 {
	if len(calculation.CalculationCategories) == 0 {
		return 0
	}

	var totalGFR float64
	var count int

	for _, calculationCategory := range calculation.CalculationCategories {
		if calculationCategory.CreatinineLevel == nil {
			continue
		}

		var category models.PatientCategory
		for _, c := range calculation.Categories {
			if c.ID == calculationCategory.CategoryID {
				category = c
				break
			}
		}

		if category.ID == 0 {
			continue
		}

		creatinine := *calculationCategory.CreatinineLevel

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

func (r *Repository) CompleteGlomerularCalculation(calculationID, moderatorID uint) error {
	calculation, err := r.GetGlomerularCalculationByID(calculationID)
	if err != nil {
		return err
	}

	gfr := r.CalculateGFR(calculation)
	now := time.Now()

	return r.db.Model(&models.GlomerularCalculation{}).
		Where("id = ?", calculationID).
		Updates(map[string]interface{}{
			"status":         models.CalculationStatusCompleted,
			"completed_at":   &now,
			"moderator_id":   moderatorID,
			"calculated_gfr": gfr,
		}).Error
}

func (r *Repository) GetDraftGlomerularCalculation(userID uint) (*models.GlomerularCalculation, error) {
	var calculation models.GlomerularCalculation
	err := r.db.Where("creator_id = ? AND status = ? AND deleted_at IS NULL", userID, models.CalculationStatusDraft).First(&calculation).Error
	if err != nil {
		return nil, err
	}
	return &calculation, nil
}

func (r *Repository) UpdateGlomerularCalculationDoctorName(calculationID uint, doctorName string) error {
	return r.db.Model(&models.GlomerularCalculation{}).
		Where("id = ?", calculationID).
		Update("doctor_name", doctorName).Error
}

func (r *Repository) UpdateGlomerularCalculationAverageAge(calculationID uint, averageAge *float64) error {
	return r.db.Model(&models.GlomerularCalculation{}).
		Where("id = ?", calculationID).
		Update("average_age", averageAge).Error
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
