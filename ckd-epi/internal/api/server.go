package api

import (
	"log"
	"time"

	"ckd-epi/internal/app/database"
	"ckd-epi/internal/app/handler"
	"ckd-epi/internal/app/repository"
	"ckd-epi/internal/app/storage"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	log.Println("Starting server")

	minioClient, err := storage.NewMinIOClient()
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}
	log.Println("MinIO client initialized")

	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := database.SeedData(minioClient); err != nil {
		log.Printf("Warning: failed to seed database: %v", err)
	}

	time.Sleep(1 * time.Second)

	repo, err := repository.NewRepository()
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}

	h := handler.NewHandler(repo, minioClient)
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	r.GET("/", h.PatientCategoriesList)
	r.GET("/categories", h.PatientCategoriesList)
	r.GET("/category/:id", h.PatientCategoryDetail)
	r.POST("/order/:id/delete", h.DeleteGFROrder)
	r.POST("/categories/:id/add", h.CreateDraftGFROrderAndAddCategory)
	r.GET("/order/:id", h.GFROrderDetail)
	r.GET("/images/:key", h.GetImage)

	r.Run() // :8080
	log.Println("Server down")
}
