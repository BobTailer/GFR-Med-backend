package handler

import (
	"ckd-epi/internal/app/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repo *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repo: r}
}

// Главная страница — список пациентов
func (h *Handler) ServicesList(ctx *gin.Context) {
	query := ctx.Query("search")
	patients := h.Repo.FilterPatients(query)
	ctx.HTML(http.StatusOK, "services.html", gin.H{
		"patients": patients,
		"search":   query,
	})
}

// Подробности пациента
func (h *Handler) PatientDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid patient ID")
		return
	}
	patient, found := h.Repo.GetPatientByID(id)
	if !found {
		ctx.String(http.StatusNotFound, "Patient not found")
		return
	}
	ctx.HTML(http.StatusOK, "patient.html", gin.H{
		"patient": patient,
	})
}

// Заявка (расчёт)
func (h *Handler) OrderDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid order ID")
		return
	}
	order, found := h.Repo.GetOrder(id)
	if !found {
		ctx.String(http.StatusNotFound, "Order not found")
		return
	}
	ctx.HTML(http.StatusOK, "order.html", gin.H{
		"order":   order,
		"patient": order.Patient,
	})
}

func (h *Handler) SelectPatient(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		h.Repo.SelectPatient(id)
	}
	ctx.Redirect(http.StatusFound, "/order/"+idStr)
}
