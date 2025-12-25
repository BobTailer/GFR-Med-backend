package handler

import (
	"ckd-epi/internal/app/auth"
	"ckd-epi/internal/app/models"
	"ckd-epi/internal/app/redis"
	"ckd-epi/internal/app/repository"
	"ckd-epi/internal/app/storage"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	Repo        *repository.Repository
	MinIOClient *storage.MinIOClient
	JWTService  *auth.JWTService
	RedisClient *redis.Client
}

func NewHandler(r *repository.Repository, minioClient *storage.MinIOClient, jwtService *auth.JWTService, redisClient *redis.Client) *Handler {
	return &Handler{
		Repo:        r,
		MinIOClient: minioClient,
		JWTService:  jwtService,
		RedisClient: redisClient,
	}
}

func (h *Handler) GetCurrentUserFromContext(ctx *gin.Context) (*models.User, error) {
	userID, exists := ctx.Get("userID")
	if !exists {
		return nil, fmt.Errorf("userID not found in context")
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		return nil, fmt.Errorf("invalid userID type in context")
	}

	user, err := h.Repo.GetUserByID(userIDUint)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *Handler) PatientCategoriesList(ctx *gin.Context) {
	query := ctx.Query("category_name")

	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	categories, err := h.Repo.GetPatientCategories(query)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	type CategoryWithImageURL struct {
		ID          uint
		Name        string
		Description string
		Gender      string
		Age         int
		ImageKey    *string
		ImageURL    string
	}

	categoriesWithURLs := make([]CategoryWithImageURL, len(categories))
	for i, cat := range categories {
		var imageURL string
		if cat.ImageKey != nil {
			imageURL = h.MinIOClient.GetPublicURL(*cat.ImageKey)
		}
		categoriesWithURLs[i] = CategoryWithImageURL{
			ID:          cat.ID,
			Name:        cat.Name,
			Description: cat.Description,
			Gender:      cat.Gender,
			Age:         cat.Age,
			ImageKey:    cat.ImageKey,
			ImageURL:    imageURL,
		}
	}

	var hasDraftCalculation bool
	var draftCalculationID uint
	var categoriesCount int
	draftCalculation, err := h.Repo.GetDraftGlomerularCalculation(user.ID)
	if err == nil && draftCalculation != nil {
		hasDraftCalculation = true
		draftCalculationID = draftCalculation.ID

		fullCalculation, err := h.Repo.GetGlomerularCalculationByID(draftCalculation.ID)
		if err == nil && fullCalculation != nil {
			categoriesCount = len(fullCalculation.CalculationCategories)
		}
	}

	ctx.HTML(http.StatusOK, "services.html", gin.H{
		"categories":          categoriesWithURLs,
		"category_name":       query,
		"hasDraftCalculation": hasDraftCalculation,
		"draftCalculationID":  draftCalculationID,
		"categoriesCount":     categoriesCount,
	})
}

func (h *Handler) PatientCategoryDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	category, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	var hasDraftCalculation bool
	var draftCalculationID uint
	var categoriesCount int
	draftCalculation, err := h.Repo.GetDraftGlomerularCalculation(user.ID)
	if err == nil && draftCalculation != nil {
		hasDraftCalculation = true
		draftCalculationID = draftCalculation.ID

		fullCalculation, err := h.Repo.GetGlomerularCalculationByID(draftCalculation.ID)
		if err == nil && fullCalculation != nil {
			categoriesCount = len(fullCalculation.CalculationCategories)
		}
	}

	var imageURL string
	if category.ImageKey != nil {
		imageURL = h.MinIOClient.GetPublicURL(*category.ImageKey)
	}

	ctx.HTML(http.StatusOK, "patient.html", gin.H{
		"category":            category,
		"imageURL":            imageURL,
		"hasDraftCalculation": hasDraftCalculation,
		"draftCalculationID":  draftCalculationID,
		"categoriesCount":     categoriesCount,
	})
}

func (h *Handler) GlomerularCalculationDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	if calculation.Status == models.CalculationStatusDeleted {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	var hasDraftCalculation bool
	var draftCalculationID uint
	var categoriesCount int
	draftCalculation, err := h.Repo.GetDraftGlomerularCalculation(user.ID)
	if err == nil && draftCalculation != nil {
		hasDraftCalculation = true
		draftCalculationID = draftCalculation.ID

		fullCalculation, err := h.Repo.GetGlomerularCalculationByID(draftCalculation.ID)
		if err == nil && fullCalculation != nil {
			categoriesCount = len(fullCalculation.CalculationCategories)
		}
	}

	type CategoryWithCalculationData struct {
		ID              uint
		Name            string
		Description     string
		Gender          string
		Age             int
		ImageKey        *string
		ImageURL        string
		CreatinineLevel *float64
		CalculatedGFR   *float64
	}

	calculationCategoryMap := make(map[uint]*models.GlomerularCalculationCategory)
	for i := range calculation.CalculationCategories {
		cc := &calculation.CalculationCategories[i]
		calculationCategoryMap[cc.CategoryID] = cc
	}

	categoriesWithData := make([]CategoryWithCalculationData, 0, len(calculation.Categories))
	for _, cat := range calculation.Categories {
		var imageURL string
		if cat.ImageKey != nil {
			imageURL = h.MinIOClient.GetPublicURL(*cat.ImageKey)
		}

		calcCat := calculationCategoryMap[cat.ID]
		var creatinineLevel *float64
		var calculatedGFR *float64
		if calcCat != nil {
			creatinineLevel = calcCat.CreatinineLevel
			calculatedGFR = calcCat.CalculatedGFR
		}

		categoriesWithData = append(categoriesWithData, CategoryWithCalculationData{
			ID:              cat.ID,
			Name:            cat.Name,
			Description:     cat.Description,
			Gender:          cat.Gender,
			Age:             cat.Age,
			ImageKey:        cat.ImageKey,
			ImageURL:        imageURL,
			CreatinineLevel: creatinineLevel,
			CalculatedGFR:   calculatedGFR,
		})
	}

	var averageAge float64
	if calculation.AverageAge != nil {
		averageAge = *calculation.AverageAge
	} else if len(calculation.Categories) > 0 {
		var totalAge int
		for _, cat := range calculation.Categories {
			totalAge += cat.Age
		}
		averageAge = float64(totalAge) / float64(len(calculation.Categories))

		avgAgePtr := &averageAge
		h.Repo.UpdateGlomerularCalculationAverageAge(calculation.ID, avgAgePtr)
	}

	ctx.HTML(http.StatusOK, "order.html", gin.H{
		"calculation":         calculation,
		"categories":          categoriesWithData,
		"averageAge":          averageAge,
		"hasDraftCalculation": hasDraftCalculation,
		"draftCalculationID":  draftCalculationID,
		"categoriesCount":     categoriesCount,
	})
}

func (h *Handler) CreateDraftGlomerularCalculationAndAddCategory(ctx *gin.Context) {
	categoryIDStr := ctx.Param("id")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	calculation, err := h.Repo.GetOrCreateDraftGlomerularCalculation(user.ID)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	err = h.Repo.AddCategoryToGlomerularCalculation(calculation.ID, uint(categoryID))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	ctx.Redirect(http.StatusFound, fmt.Sprintf("/glomerular-calculation/%d", calculation.ID))
}

func (h *Handler) DeleteGlomerularCalculation(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	err = h.Repo.DeleteGlomerularCalculation(uint(id))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	ctx.Redirect(http.StatusFound, "/patient-categories")
}

func (h *Handler) GetImage(ctx *gin.Context) {
	imageKey := ctx.Param("key")
	if imageKey == "" {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	ctxReq, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	objInfo, err := h.MinIOClient.StatObject(ctxReq, imageKey)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}

	obj, err := h.MinIOClient.GetObject(ctxReq, imageKey)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/patient-categories")
		return
	}
	defer obj.Close()

	contentType := objInfo.ContentType
	if contentType == "" {
		if len(imageKey) > 4 {
			ext := imageKey[len(imageKey)-4:]
			switch ext {
			case ".png":
				contentType = "image/png"
			case ".jpg":
				contentType = "image/jpeg"
			default:
				if len(imageKey) > 5 && imageKey[len(imageKey)-5:] == "jpeg" {
					contentType = "image/jpeg"
				} else {
					contentType = "application/octet-stream"
				}
			}
		} else {
			contentType = "application/octet-stream"
		}
	}

	ctx.Header("Content-Type", contentType)
	ctx.Header("Cache-Control", "public, max-age=3600")

	_, err = io.Copy(ctx.Writer, obj)
	if err != nil {
		log.Printf("Error copying image to response: %v", err)
		return
	}
}

func (h *Handler) Handle404(ctx *gin.Context) {
	ctx.Redirect(http.StatusFound, "/patient-categories")
}

func validateStatusTransition(current models.CalculationStatus, newStatus models.CalculationStatus, isCreator bool, isModerator bool) error {
	if isCreator {
		if current == models.CalculationStatusDraft && newStatus == models.CalculationStatusDeleted {
			return nil
		}
		if current == models.CalculationStatusDraft && newStatus == models.CalculationStatusFormed {
			return nil
		}
	}
	if isModerator {
		if current == models.CalculationStatusFormed && newStatus == models.CalculationStatusCompleted {
			return nil
		}
		if current == models.CalculationStatusFormed && newStatus == models.CalculationStatusRejected {
			return nil
		}
	}
	return fmt.Errorf("invalid status transition from %s to %s", current, newStatus)
}

// GetPatientCategoriesAPI godoc
// @Summary Get list of patient categories
// @Description Returns list of patient categories with optional filtering by name
// @Tags Patient Categories
// @Accept json
// @Produce json
// @Param category_name query string false "Search by category name"
// @Success 200 {object} models.PatientCategoryListResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /patient-categories [get]
func (h *Handler) GetPatientCategoriesAPI(ctx *gin.Context) {
	query := ctx.Query("category_name")

	categories, err := h.Repo.GetPatientCategories(query)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	responses := make([]models.PatientCategoryResponse, len(categories))
	for i, cat := range categories {
		var imageURL *string
		if cat.ImageKey != nil {
			url := h.MinIOClient.GetPublicURL(*cat.ImageKey)
			imageURL = &url
		}
		responses[i] = models.PatientCategoryResponse{
			ID:          cat.ID,
			Name:        cat.Name,
			Description: cat.Description,
			Gender:      cat.Gender,
			Age:         cat.Age,
			ImageURL:    imageURL,
		}
	}

	ctx.JSON(http.StatusOK, models.PatientCategoryListResponse{Categories: responses})
}

// GetPatientCategoryAPI godoc
// @Summary Get patient category by ID
// @Description Returns a single patient category by its ID
// @Tags Patient Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} models.PatientCategoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /patient-categories/{id} [get]
func (h *Handler) GetPatientCategoryAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	category, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "category not found"})
		return
	}

	var imageURL *string
	if category.ImageKey != nil {
		url := h.MinIOClient.GetPublicURL(*category.ImageKey)
		imageURL = &url
	}

	ctx.JSON(http.StatusOK, models.PatientCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		Gender:      category.Gender,
		Age:         category.Age,
		ImageURL:    imageURL,
	})
}

// CreatePatientCategoryAPI godoc
// @Summary Create new patient category
// @Description Creates a new patient category
// @Tags Patient Categories
// @Accept json
// @Produce json
// @Param request body models.CreatePatientCategoryRequest true "Category data"
// @Success 201 {object} models.PatientCategoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /patient-categories [post]
func (h *Handler) CreatePatientCategoryAPI(ctx *gin.Context) {
	var req models.CreatePatientCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	category := models.PatientCategory{
		Name:        req.Name,
		Description: req.Description,
		Gender:      req.Gender,
		Age:         req.Age,
		IsDeleted:   false,
	}

	if err := h.Repo.CreatePatientCategory(&category); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	var imageURL *string
	if category.ImageKey != nil {
		url := h.MinIOClient.GetPublicURL(*category.ImageKey)
		imageURL = &url
	}

	ctx.JSON(http.StatusCreated, models.PatientCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		Gender:      category.Gender,
		Age:         category.Age,
		ImageURL:    imageURL,
	})
}

// UpdatePatientCategoryAPI godoc
// @Summary Update patient category
// @Description Updates an existing patient category
// @Tags Patient Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Param request body models.UpdatePatientCategoryRequest true "Category data"
// @Success 200 {object} models.PatientCategoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /patient-categories/{id} [put]
func (h *Handler) UpdatePatientCategoryAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var req models.UpdatePatientCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}
	if req.Age != nil {
		updates["age"] = *req.Age
	}

	if err := h.Repo.UpdatePatientCategory(uint(id), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	category, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "category not found"})
		return
	}

	var imageURL *string
	if category.ImageKey != nil {
		url := h.MinIOClient.GetPublicURL(*category.ImageKey)
		imageURL = &url
	}

	ctx.JSON(http.StatusOK, models.PatientCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		Gender:      category.Gender,
		Age:         category.Age,
		ImageURL:    imageURL,
	})
}

// DeletePatientCategoryAPI godoc
// @Summary Delete patient category
// @Description Deletes a patient category (logical delete)
// @Tags Patient Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /patient-categories/{id} [delete]
func (h *Handler) DeletePatientCategoryAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	category, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "category not found"})
		return
	}

	if category.ImageKey != nil && *category.ImageKey != "" {
		ctxReq, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		_ = h.MinIOClient.DeleteObject(ctxReq, *category.ImageKey)
	}

	if err := h.Repo.DeletePatientCategory(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddCategoryToCalculationAPI godoc
// @Summary Add category to calculation
// @Description Adds a patient category to the current user's draft calculation
// @Tags Patient Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /patient-categories/{id}/add-to-calculation [post]
func (h *Handler) AddCategoryToCalculationAPI(ctx *gin.Context) {
	categoryIDStr := ctx.Param("id")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid category id"})
		return
	}

	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	calculation, err := h.Repo.GetOrCreateDraftGlomerularCalculation(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	err = h.Repo.AddCategoryToGlomerularCalculation(calculation.ID, uint(categoryID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"calculation_id": calculation.ID,
		"category_id":    categoryID,
		"message":        "category added to calculation",
	})
}

// UploadPatientCategoryImageAPI godoc
// @Summary Upload patient category image
// @Description Uploads an image for a patient category to MinIO
// @Tags Patient Categories
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Category ID"
// @Param image formData file true "Image file"
// @Success 200 {object} models.PatientCategoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /patient-categories/{id}/image [post]
func (h *Handler) UploadPatientCategoryImageAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	category, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "category not found"})
		return
	}

	file, err := ctx.FormFile("image")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "image file required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	defer src.Close()

	filename := file.Filename
	objectKey := storage.GenerateObjectKey(filename)

	if category.ImageKey != nil && *category.ImageKey != "" {
		ctxReq, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		_ = h.MinIOClient.DeleteObject(ctxReq, *category.ImageKey)
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		if strings.HasSuffix(strings.ToLower(filename), ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(filename), ".jpg") || strings.HasSuffix(strings.ToLower(filename), ".jpeg") {
			contentType = "image/jpeg"
		} else {
			contentType = "application/octet-stream"
		}
	}

	ctxReq, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()

	err = h.MinIOClient.UploadFromReader(ctxReq, objectKey, src, file.Size, contentType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	imageURL := h.MinIOClient.GetPublicURL(objectKey)

	updates := map[string]interface{}{
		"image_key": objectKey,
		"image_url": &imageURL,
	}

	if err := h.Repo.UpdatePatientCategory(uint(id), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	updatedCategory, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	var responseImageURL *string
	if updatedCategory.ImageKey != nil {
		url := h.MinIOClient.GetPublicURL(*updatedCategory.ImageKey)
		responseImageURL = &url
	}

	ctx.JSON(http.StatusOK, models.PatientCategoryResponse{
		ID:          updatedCategory.ID,
		Name:        updatedCategory.Name,
		Description: updatedCategory.Description,
		Gender:      updatedCategory.Gender,
		Age:         updatedCategory.Age,
		ImageURL:    responseImageURL,
	})
}

// GetCartIconAPI godoc
// @Summary Get cart icon info
// @Description Returns the current user's draft calculation ID and category count
// @Tags Calculations
// @Accept json
// @Produce json
// @Success 200 {object} models.CartIconResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /cart-icon [get]
func (h *Handler) GetCartIconAPI(ctx *gin.Context) {
	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	draft, err := h.Repo.GetDraftGlomerularCalculation(user.ID)
	if err != nil {
		ctx.JSON(http.StatusOK, models.CartIconResponse{ID: nil, Count: 0})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(draft.ID)
	if err != nil {
		ctx.JSON(http.StatusOK, models.CartIconResponse{ID: nil, Count: 0})
		return
	}

	ctx.JSON(http.StatusOK, models.CartIconResponse{
		ID:    &calculation.ID,
		Count: len(calculation.CalculationCategories),
	})
}

// GetGlomerularCalculationsAPI godoc
// @Summary Get list of glomerular calculations
// @Description Returns list of glomerular calculations with optional filtering by status and date range. For non-moderators, returns only their own calculations.
// @Tags Calculations
// @Accept json
// @Produce json
// @Param status query string false "Filter by status"
// @Param formed_from query string false "Filter by formation date from (RFC3339)"
// @Param formed_to query string false "Filter by formation date to (RFC3339)"
// @Success 200 {object} models.GlomerularCalculationListResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations [get]
func (h *Handler) GetGlomerularCalculationsAPI(ctx *gin.Context) {
	status := ctx.Query("status")
	formedFromStr := ctx.Query("formed_from")
	formedToStr := ctx.Query("formed_to")

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	var formedFrom *time.Time
	if formedFromStr != "" {
		parsed, err := time.Parse(time.RFC3339, formedFromStr)
	if err == nil {
			formedFrom = &parsed
		}
	}

	var formedTo *time.Time
	if formedToStr != "" {
		parsed, err := time.Parse(time.RFC3339, formedToStr)
	if err == nil {
			formedTo = &parsed
		}
	}

	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	var creatorID *uint
	if !user.IsModerator {
		creatorID = &user.ID
	}

	calculations, err := h.Repo.GetGlomerularCalculationsList(statusPtr, formedFrom, formedTo, creatorID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	items := make([]models.GlomerularCalculationListItem, len(calculations))
	for i, calc := range calculations {
		creator, _ := h.Repo.GetUserByID(calc.CreatorID)
		var creatorUsername string
		if creator != nil {
			creatorUsername = creator.Username
		}

		var moderatorUsername *string
		if calc.ModeratorID != nil {
			moderator, _ := h.Repo.GetUserByID(*calc.ModeratorID)
			if moderator != nil {
				moderatorUsername = &moderator.Username
			}
		}

		categoriesWithGFR := 0
		for _, cc := range calc.CalculationCategories {
			if cc.CalculatedGFR != nil {
				categoriesWithGFR++
			}
		}

		items[i] = models.GlomerularCalculationListItem{
			ID:                calc.ID,
			Status:            calc.Status,
			CreatedAt:         calc.CreatedAt.Format(time.RFC3339),
			CreatorID:         calc.CreatorID,
			CreatorUsername:   creatorUsername,
			DoctorName:        calc.DoctorName,
			AverageAge:        calc.AverageAge,
			FormedAt:          formatTime(calc.FormedAt),
			CompletedAt:       formatTime(calc.CompletedAt),
			ModeratorID:       calc.ModeratorID,
			ModeratorUsername: moderatorUsername,
			CalculatedGFR:     calc.CalculatedGFR,
			CategoriesWithGFR: categoriesWithGFR,
		}
	}

	ctx.JSON(http.StatusOK, models.GlomerularCalculationListResponse{Calculations: items	})
}

// GetGlomerularCalculationAPI godoc
// @Summary Get glomerular calculation by ID
// @Description Returns a single glomerular calculation with all its categories
// @Tags Calculations
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Success 200 {object} models.GlomerularCalculationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id} [get]
func (h *Handler) GetGlomerularCalculationAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	if calculation.Status == models.CalculationStatusDeleted {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	creator, _ := h.Repo.GetUserByID(calculation.CreatorID)
	var creatorUsername *string
	if creator != nil {
		creatorUsername = &creator.Username
	}

	var moderatorUsername *string
	if calculation.ModeratorID != nil {
		moderator, _ := h.Repo.GetUserByID(*calculation.ModeratorID)
		if moderator != nil {
			moderatorUsername = &moderator.Username
		}
	}

	categoryMap := make(map[uint]*models.GlomerularCalculationCategory)
	for i := range calculation.CalculationCategories {
		cc := &calculation.CalculationCategories[i]
		categoryMap[cc.CategoryID] = cc
	}

	categories := make([]models.GlomerularCalculationCategoryResponse, 0, len(calculation.Categories))
	for _, cat := range calculation.Categories {
		var imageURL *string
		if cat.ImageKey != nil {
			url := h.MinIOClient.GetPublicURL(*cat.ImageKey)
			imageURL = &url
		}

		cc := categoryMap[cat.ID]
		categoryResp := models.GlomerularCalculationCategoryResponse{
			CategoryID: cat.ID,
			Category: models.PatientCategoryResponse{
				ID:          cat.ID,
				Name:        cat.Name,
				Description: cat.Description,
				Gender:      cat.Gender,
				Age:         cat.Age,
				ImageURL:    imageURL,
			},
		}

		if cc != nil {
			categoryResp.CreatinineLevel = cc.CreatinineLevel
			categoryResp.CalculatedGFR = cc.CalculatedGFR
		}

		categories = append(categories, categoryResp)
	}

	ctx.JSON(http.StatusOK, models.GlomerularCalculationResponse{
		ID:                calculation.ID,
		Status:            calculation.Status,
		CreatedAt:         calculation.CreatedAt.Format(time.RFC3339),
		CreatorID:         calculation.CreatorID,
		CreatorUsername:   creatorUsername,
		DoctorName:        calculation.DoctorName,
		AverageAge:        calculation.AverageAge,
		FormedAt:          formatTime(calculation.FormedAt),
		CompletedAt:       formatTime(calculation.CompletedAt),
		ModeratorID:       calculation.ModeratorID,
		ModeratorUsername: moderatorUsername,
		CalculatedGFR:     calculation.CalculatedGFR,
		Categories:        categories,
	})
}

// UpdateGlomerularCalculationAPI godoc
// @Summary Update glomerular calculation
// @Description Updates fields of a glomerular calculation
// @Tags Calculations
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Param request body models.UpdateGlomerularCalculationRequest true "Calculation data"
// @Success 200 {object} models.GlomerularCalculationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id} [put]
func (h *Handler) UpdateGlomerularCalculationAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var req models.UpdateGlomerularCalculationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.DoctorName != nil {
		updates["doctor_name"] = *req.DoctorName
	}

	if err := h.Repo.UpdateGlomerularCalculation(uint(id), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	creator, _ := h.Repo.GetUserByID(calculation.CreatorID)
	var creatorUsername *string
	if creator != nil {
		creatorUsername = &creator.Username
	}

	var moderatorUsername *string
	if calculation.ModeratorID != nil {
		moderator, _ := h.Repo.GetUserByID(*calculation.ModeratorID)
		if moderator != nil {
			moderatorUsername = &moderator.Username
		}
	}

	categoryMap := make(map[uint]*models.GlomerularCalculationCategory)
	for i := range calculation.CalculationCategories {
		cc := &calculation.CalculationCategories[i]
		categoryMap[cc.CategoryID] = cc
	}

	categories := make([]models.GlomerularCalculationCategoryResponse, 0, len(calculation.Categories))
	for _, cat := range calculation.Categories {
		var imageURL *string
		if cat.ImageKey != nil {
			url := h.MinIOClient.GetPublicURL(*cat.ImageKey)
			imageURL = &url
		}

		cc := categoryMap[cat.ID]
		categoryResp := models.GlomerularCalculationCategoryResponse{
			CategoryID: cat.ID,
			Category: models.PatientCategoryResponse{
				ID:          cat.ID,
				Name:        cat.Name,
				Description: cat.Description,
				Gender:      cat.Gender,
				Age:         cat.Age,
				ImageURL:    imageURL,
			},
		}

		if cc != nil {
			categoryResp.CreatinineLevel = cc.CreatinineLevel
			categoryResp.CalculatedGFR = cc.CalculatedGFR
		}

		categories = append(categories, categoryResp)
	}

	ctx.JSON(http.StatusOK, models.GlomerularCalculationResponse{
		ID:                calculation.ID,
		Status:            calculation.Status,
		CreatedAt:         calculation.CreatedAt.Format(time.RFC3339),
		CreatorID:         calculation.CreatorID,
		CreatorUsername:   creatorUsername,
		DoctorName:        calculation.DoctorName,
		AverageAge:        calculation.AverageAge,
		FormedAt:          formatTime(calculation.FormedAt),
		CompletedAt:       formatTime(calculation.CompletedAt),
		ModeratorID:       calculation.ModeratorID,
		ModeratorUsername: moderatorUsername,
		CalculatedGFR:     calculation.CalculatedGFR,
		Categories:        categories,
	})
}

// FormGlomerularCalculationAPI godoc
// @Summary Form glomerular calculation
// @Description Forms a draft calculation (changes status to 'formed'). Only creator can form their draft.
// @Tags Calculations
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Success 200 {object} models.GlomerularCalculationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id}/form [put]
func (h *Handler) FormGlomerularCalculationAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	if calculation.CreatorID != user.ID {
		ctx.JSON(http.StatusForbidden, models.ErrorResponse{Error: "only creator can form calculation"})
		return
	}

	if err := validateStatusTransition(calculation.Status, models.CalculationStatusFormed, true, false); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.Repo.FormGlomerularCalculation(uint(id)); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	updated, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	creator, _ := h.Repo.GetUserByID(updated.CreatorID)
	var creatorUsername *string
	if creator != nil {
		creatorUsername = &creator.Username
	}

	var moderatorUsername *string
	if updated.ModeratorID != nil {
		moderator, _ := h.Repo.GetUserByID(*updated.ModeratorID)
		if moderator != nil {
			moderatorUsername = &moderator.Username
		}
	}

	categoryMap := make(map[uint]*models.GlomerularCalculationCategory)
	for i := range updated.CalculationCategories {
		cc := &updated.CalculationCategories[i]
		categoryMap[cc.CategoryID] = cc
	}

	categories := make([]models.GlomerularCalculationCategoryResponse, 0, len(updated.Categories))
	for _, cat := range updated.Categories {
		var imageURL *string
		if cat.ImageKey != nil {
			url := h.MinIOClient.GetPublicURL(*cat.ImageKey)
			imageURL = &url
		}

		cc := categoryMap[cat.ID]
		categoryResp := models.GlomerularCalculationCategoryResponse{
			CategoryID: cat.ID,
			Category: models.PatientCategoryResponse{
				ID:          cat.ID,
				Name:        cat.Name,
				Description: cat.Description,
				Gender:      cat.Gender,
				Age:         cat.Age,
				ImageURL:    imageURL,
			},
		}

		if cc != nil {
			categoryResp.CreatinineLevel = cc.CreatinineLevel
			categoryResp.CalculatedGFR = cc.CalculatedGFR
		}

		categories = append(categories, categoryResp)
	}

	ctx.JSON(http.StatusOK, models.GlomerularCalculationResponse{
		ID:                updated.ID,
		Status:            updated.Status,
		CreatedAt:         updated.CreatedAt.Format(time.RFC3339),
		CreatorID:         updated.CreatorID,
		CreatorUsername:   creatorUsername,
		DoctorName:        updated.DoctorName,
		AverageAge:        updated.AverageAge,
		FormedAt:          formatTime(updated.FormedAt),
		CompletedAt:       formatTime(updated.CompletedAt),
		ModeratorID:       updated.ModeratorID,
		ModeratorUsername: moderatorUsername,
		CalculatedGFR:     updated.CalculatedGFR,
		Categories:        categories,
	})
}

// CompleteGlomerularCalculationAPI godoc
// @Summary Complete or reject glomerular calculation
// @Description Completes or rejects a formed calculation. Only moderator can complete/reject.
// @Tags Calculations
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Param request body models.CompleteGlomerularCalculationRequest true "Action: 'complete' or 'reject'"
// @Success 200 {object} models.GlomerularCalculationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id}/complete [put]
func (h *Handler) CompleteGlomerularCalculationAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	var req models.CompleteGlomerularCalculationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	if !user.IsModerator {
		ctx.JSON(http.StatusForbidden, models.ErrorResponse{Error: "only moderator can complete/reject calculation"})
		return
	}

	if req.Action == "complete" {
		if err := validateStatusTransition(calculation.Status, models.CalculationStatusCompleted, false, true); err != nil {
			ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}

		if err := h.Repo.CompleteGlomerularCalculationWithCategories(uint(id), user.ID); err != nil {
			ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
	} else if req.Action == "reject" {
		if err := validateStatusTransition(calculation.Status, models.CalculationStatusRejected, false, true); err != nil {
			ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}

		if err := h.Repo.RejectGlomerularCalculation(uint(id), user.ID); err != nil {
			ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
	} else {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "action must be 'complete' or 'reject'"})
		return
	}

	updated, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	creator, _ := h.Repo.GetUserByID(updated.CreatorID)
	var creatorUsername *string
	if creator != nil {
		creatorUsername = &creator.Username
	}

	var moderatorUsername *string
	if updated.ModeratorID != nil {
		moderator, _ := h.Repo.GetUserByID(*updated.ModeratorID)
		if moderator != nil {
			moderatorUsername = &moderator.Username
		}
	}

	categoryMap := make(map[uint]*models.GlomerularCalculationCategory)
	for i := range updated.CalculationCategories {
		cc := &updated.CalculationCategories[i]
		categoryMap[cc.CategoryID] = cc
	}

	categories := make([]models.GlomerularCalculationCategoryResponse, 0, len(updated.Categories))
	for _, cat := range updated.Categories {
		var imageURL *string
		if cat.ImageKey != nil {
			url := h.MinIOClient.GetPublicURL(*cat.ImageKey)
			imageURL = &url
		}

		cc := categoryMap[cat.ID]
		categoryResp := models.GlomerularCalculationCategoryResponse{
			CategoryID: cat.ID,
			Category: models.PatientCategoryResponse{
				ID:          cat.ID,
				Name:        cat.Name,
				Description: cat.Description,
				Gender:      cat.Gender,
				Age:         cat.Age,
				ImageURL:    imageURL,
			},
		}

		if cc != nil {
			categoryResp.CreatinineLevel = cc.CreatinineLevel
			categoryResp.CalculatedGFR = cc.CalculatedGFR
		}

		categories = append(categories, categoryResp)
	}

	ctx.JSON(http.StatusOK, models.GlomerularCalculationResponse{
		ID:                updated.ID,
		Status:            updated.Status,
		CreatedAt:         updated.CreatedAt.Format(time.RFC3339),
		CreatorID:         updated.CreatorID,
		CreatorUsername:   creatorUsername,
		DoctorName:        updated.DoctorName,
		AverageAge:        updated.AverageAge,
		FormedAt:          formatTime(updated.FormedAt),
		CompletedAt:       formatTime(updated.CompletedAt),
		ModeratorID:       updated.ModeratorID,
		ModeratorUsername: moderatorUsername,
		CalculatedGFR:     updated.CalculatedGFR,
		Categories:        categories,
	})
}

// DeleteGlomerularCalculationAPI godoc
// @Summary Delete glomerular calculation
// @Description Deletes a draft calculation (logical delete). Only creator can delete their draft.
// @Tags Calculations
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id} [delete]
func (h *Handler) DeleteGlomerularCalculationAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid id"})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	if calculation.CreatorID != user.ID {
		ctx.JSON(http.StatusForbidden, models.ErrorResponse{Error: "only creator can delete calculation"})
		return
	}

	if err := validateStatusTransition(calculation.Status, models.CalculationStatusDeleted, true, false); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.Repo.DeleteGlomerularCalculation(uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UpdateGlomerularCalculationCategoryAPI godoc
// @Summary Update calculation category
// @Description Updates creatinine level for a category in a calculation
// @Tags Calculation Categories
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Param categoryId path int true "Category ID"
// @Param request body models.UpdateGlomerularCalculationCategoryRequest true "Creatinine level"
// @Success 200 {object} models.GlomerularCalculationCategoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id}/categories/{categoryId} [put]
func (h *Handler) UpdateGlomerularCalculationCategoryAPI(ctx *gin.Context) {
	calculationIDStr := ctx.Param("id")
	calculationID, err := strconv.ParseUint(calculationIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid calculation id"})
		return
	}

	categoryIDStr := ctx.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid category id"})
		return
	}

	var req models.UpdateGlomerularCalculationCategoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.CreatinineLevel != nil {
		updates["creatinine_level"] = *req.CreatinineLevel
	}

	if err := h.Repo.UpdateGlomerularCalculationCategory(uint(calculationID), uint(categoryID), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	calculation, err := h.Repo.GetGlomerularCalculationByID(uint(calculationID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "calculation not found"})
		return
	}

	categoryMap := make(map[uint]*models.GlomerularCalculationCategory)
	for i := range calculation.CalculationCategories {
		cc := &calculation.CalculationCategories[i]
		categoryMap[cc.CategoryID] = cc
	}

	var foundCategory *models.PatientCategory
	for _, cat := range calculation.Categories {
		if cat.ID == uint(categoryID) {
			foundCategory = &cat
			break
		}
	}

	if foundCategory == nil {
		ctx.JSON(http.StatusNotFound, models.ErrorResponse{Error: "category not found in calculation"})
		return
	}

	var imageURL *string
	if foundCategory.ImageKey != nil {
		url := h.MinIOClient.GetPublicURL(*foundCategory.ImageKey)
		imageURL = &url
	}

	cc := categoryMap[uint(categoryID)]
	categoryResp := models.GlomerularCalculationCategoryResponse{
		CategoryID: foundCategory.ID,
		Category: models.PatientCategoryResponse{
			ID:          foundCategory.ID,
			Name:        foundCategory.Name,
			Description: foundCategory.Description,
			Gender:      foundCategory.Gender,
			Age:         foundCategory.Age,
			ImageURL:    imageURL,
		},
	}

	if cc != nil {
		categoryResp.CreatinineLevel = cc.CreatinineLevel
		categoryResp.CalculatedGFR = cc.CalculatedGFR
	}

	ctx.JSON(http.StatusOK, categoryResp)
}

// DeleteGlomerularCalculationCategoryAPI godoc
// @Summary Delete category from calculation
// @Description Removes a category from a calculation
// @Tags Calculation Categories
// @Accept json
// @Produce json
// @Param id path int true "Calculation ID"
// @Param categoryId path int true "Category ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /glomerular-calculations/{id}/categories/{categoryId} [delete]
func (h *Handler) DeleteGlomerularCalculationCategoryAPI(ctx *gin.Context) {
	calculationIDStr := ctx.Param("id")
	calculationID, err := strconv.ParseUint(calculationIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid calculation id"})
		return
	}

	categoryIDStr := ctx.Param("categoryId")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid category id"})
		return
	}

	if err := h.Repo.DeleteGlomerularCalculationCategory(uint(calculationID), uint(categoryID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// RegisterUserAPI godoc
// @Summary Register new user
// @Description Registers a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param request body models.RegisterUserRequest true "User registration data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /users/register [post]
func (h *Handler) RegisterUserAPI(ctx *gin.Context) {
	var req models.RegisterUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	existing, _ := h.Repo.GetUserByUsername(req.Username)
	if existing != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "username already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	user := models.User{
		Username:    req.Username,
		Password:    string(hashedPassword),
		IsModerator: false,
	}

	if err := h.Repo.CreateUser(&user); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, models.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		IsModerator: user.IsModerator,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}

// GetUserMeAPI godoc
// @Summary Get current user info
// @Description Returns information about the currently authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} models.UserResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /users/me [get]
func (h *Handler) GetUserMeAPI(ctx *gin.Context) {
	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	ctx.JSON(http.StatusOK, models.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		IsModerator: user.IsModerator,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	})
}

// UpdateUserMeAPI godoc
// @Summary Update current user
// @Description Updates information of the currently authenticated user
// @Tags Users
// @Accept json
// @Produce json
// @Param request body models.UpdateUserRequest true "User update data"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /users/me [put]
func (h *Handler) UpdateUserMeAPI(ctx *gin.Context) {
	user, err := h.GetCurrentUserFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "unauthorized"})
		return
	}

	var req models.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Username != nil {
		existing, _ := h.Repo.GetUserByUsername(*req.Username)
		if existing != nil && existing.ID != user.ID {
			ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "username already exists"})
			return
		}
		updates["username"] = *req.Username
	}
	if req.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
			return
		}
		updates["password"] = string(hashedPassword)
	}

	if err := h.Repo.UpdateUser(user.ID, updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	updated, err := h.Repo.GetUserByID(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, models.UserResponse{
		ID:          updated.ID,
		Username:    updated.Username,
		IsModerator: updated.IsModerator,
		CreatedAt:   updated.CreatedAt.Format(time.RFC3339),
	})
}

// AuthUserAPI godoc
// @Summary Authenticate user
// @Description Authenticate user with username and password, returns JWT token
// @Tags Users
// @Accept json
// @Produce json
// @Param request body models.AuthRequest true "Authentication credentials"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /users/auth [post]
func (h *Handler) AuthUserAPI(ctx *gin.Context) {
	var req models.AuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.Repo.GetUserByUsername(req.Username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid username or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid username or password"})
		return
	}

	token, err := h.JWTService.GenerateToken(user.ID, user.Username, user.IsModerator)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	err = h.RedisClient.StoreSession(ctx.Request.Context(), token, user.ID, 24*time.Hour)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.SetCookie("session_token", token, int(24*time.Hour.Seconds()), "/", "", false, true)

	ctx.JSON(http.StatusOK, models.AuthResponse{
		Token: token,
		User: models.UserResponse{
			ID:          user.ID,
			Username:    user.Username,
			IsModerator: user.IsModerator,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
	})
}

// LogoutUserAPI godoc
// @Summary Logout user
// @Description Logout user by adding JWT token to blacklist
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} models.LogoutResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /users/logout [post]
func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	jwtStr := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(jwtStr, jwtPrefix) {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid authorization header"})
		return
	}

	jwtStr = jwtStr[len(jwtPrefix):]

	claims, err := h.JWTService.ParseToken(jwtStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid token"})
		return
	}

	expiresAt := claims.ExpiresAt
	if expiresAt == nil {
		ctx.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "token has no expiration"})
		return
	}

	ttl := time.Until(expiresAt.Time)
	if ttl <= 0 {
		ctx.JSON(http.StatusOK, models.LogoutResponse{Message: "Logged out"})
		return
	}

	err = h.RedisClient.WriteJWTToBlacklist(ctx.Request.Context(), jwtStr, ttl)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	err = h.RedisClient.DeleteSession(ctx.Request.Context(), jwtStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx.SetCookie("session_token", "", -1, "/", "", false, true)

	ctx.JSON(http.StatusOK, models.LogoutResponse{Message: "Logged out"})
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}
