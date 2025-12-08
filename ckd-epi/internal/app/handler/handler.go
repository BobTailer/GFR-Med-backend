package handler

import (
	"ckd-epi/internal/app/models"
	"ckd-epi/internal/app/repository"
	"ckd-epi/internal/app/storage"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repo        *repository.Repository
	MinIOClient *storage.MinIOClient
}

func NewHandler(r *repository.Repository, minioClient *storage.MinIOClient) *Handler {
	return &Handler{
		Repo:        r,
		MinIOClient: minioClient,
	}
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
