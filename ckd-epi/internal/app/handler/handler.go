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
	Repo       *repository.Repository
	MinIOClient *storage.MinIOClient
}

func NewHandler(r *repository.Repository, minioClient *storage.MinIOClient) *Handler {
	return &Handler{
		Repo:       r,
		MinIOClient: minioClient,
	}
}

func (h *Handler) PatientCategoriesList(ctx *gin.Context) {
	query := ctx.Query("search")
	
	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	
	categories, err := h.Repo.GetPatientCategories(query)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to get categories")
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
	
	var hasDraftOrder bool
	var draftOrderID uint
	var categoriesCount int
	draftOrder, err := h.Repo.GetDraftGFROrder(user.ID)
	if err == nil && draftOrder != nil {
		hasDraftOrder = true
		draftOrderID = draftOrder.ID
		
		fullOrder, err := h.Repo.GetGFROrderByID(draftOrder.ID)
		if err == nil && fullOrder != nil {
			categoriesCount = len(fullOrder.OrderCategories)
		}
	}
	
	ctx.HTML(http.StatusOK, "services.html", gin.H{
		"categories":      categoriesWithURLs,
		"search":          query,
		"hasDraftOrder":   hasDraftOrder,
		"draftOrderID":    draftOrderID,
		"categoriesCount": categoriesCount,
	})
}

func (h *Handler) PatientCategoryDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid category ID")
		return
	}
	
	category, err := h.Repo.GetPatientCategoryByID(uint(id))
	if err != nil {
		ctx.String(http.StatusNotFound, "Category not found")
		return
	}
	
	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	
	var hasDraftOrder bool
	var draftOrderID uint
	var categoriesCount int
	draftOrder, err := h.Repo.GetDraftGFROrder(user.ID)
	if err == nil && draftOrder != nil {
		hasDraftOrder = true
		draftOrderID = draftOrder.ID
		
		fullOrder, err := h.Repo.GetGFROrderByID(draftOrder.ID)
		if err == nil && fullOrder != nil {
			categoriesCount = len(fullOrder.OrderCategories)
		}
	}
	
	var imageURL string
	if category.ImageKey != nil {
		imageURL = h.MinIOClient.GetPublicURL(*category.ImageKey)
	}
	
	ctx.HTML(http.StatusOK, "patient.html", gin.H{
		"category":        category,
		"imageURL":        imageURL,
		"hasDraftOrder":   hasDraftOrder,
		"draftOrderID":    draftOrderID,
		"categoriesCount": categoriesCount,
	})
}

func (h *Handler) GFROrderDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid order ID")
		return
	}
	
	gfrOrder, err := h.Repo.GetGFROrderByID(uint(id))
	if err != nil {
		ctx.String(http.StatusNotFound, "Order not found")
		return
	}
	
	if gfrOrder.Status == models.OrderStatusDeleted {
		ctx.String(http.StatusNotFound, "Order not found")
		return
	}

	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	
	var hasDraftOrder bool
	var draftOrderID uint
	var categoriesCount int
	draftOrder, err := h.Repo.GetDraftGFROrder(user.ID)
	if err == nil && draftOrder != nil {
		hasDraftOrder = true
		draftOrderID = draftOrder.ID
		
		fullOrder, err := h.Repo.GetGFROrderByID(draftOrder.ID)
		if err == nil && fullOrder != nil {
			categoriesCount = len(fullOrder.OrderCategories)
		}
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
	
	categoriesWithURLs := make([]CategoryWithImageURL, len(gfrOrder.Categories))
	for i, cat := range gfrOrder.Categories {
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

	ctx.HTML(http.StatusOK, "order.html", gin.H{
		"order":           gfrOrder,
		"categories":      categoriesWithURLs,
		"orderCategories": gfrOrder.OrderCategories,
		"hasDraftOrder":   hasDraftOrder,
		"draftOrderID":    draftOrderID,
		"categoriesCount": categoriesCount,
	})
}

func (h *Handler) CreateDraftGFROrderAndAddCategory(ctx *gin.Context) {
	categoryIDStr := ctx.Param("id")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid category ID")
		return
	}
	
	user, err := h.Repo.GetDefaultUser()
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	
	gfrOrder, err := h.Repo.GetOrCreateDraftGFROrder(user.ID)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to create order")
		return
	}
	
	err = h.Repo.AddCategoryToGFROrder(gfrOrder.ID, uint(categoryID))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to add category to order")
		return
	}
	
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/order/%d", gfrOrder.ID))
}

func (h *Handler) DeleteGFROrder(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid order ID")
		return
	}
	
	err = h.Repo.DeleteGFROrder(uint(id))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to delete order")
		return
	}
	
	ctx.Redirect(http.StatusFound, "/categories")
}

func (h *Handler) GetImage(ctx *gin.Context) {
	imageKey := ctx.Param("key")
	if imageKey == "" {
		ctx.String(http.StatusBadRequest, "Image key is required")
		return
	}

	ctxReq, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()

	objInfo, err := h.MinIOClient.StatObject(ctxReq, imageKey)
	if err != nil {
		ctx.String(http.StatusNotFound, "Image not found")
		return
	}

	obj, err := h.MinIOClient.GetObject(ctxReq, imageKey)
	if err != nil {
		ctx.String(http.StatusNotFound, "Image not found")
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
