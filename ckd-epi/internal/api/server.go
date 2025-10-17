package api

import (
	"log"

	"ckd-epi/internal/app/handler"
	"ckd-epi/internal/app/repository"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	log.Println("Starting server")

	repo, _ := repository.NewRepository()
	h := handler.NewHandler(repo)
	r := gin.Default()

	// Подключаем шаблоны
	r.LoadHTMLGlob("templates/*")

	// Подключаем статику (картинки, css)
	r.Static("/static", "./resources")

	// Маршруты
	r.GET("/", h.ServicesList)
	r.GET("/services", h.ServicesList)
	r.GET("/patient/:id", h.PatientDetail)
	r.GET("/order/:id", h.OrderDetail)
	// Для кнопки "Выбрать"
	r.POST("/select/:id", h.SelectPatient)

	r.Run() // :8080
	log.Println("Server down")
}
