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
	r.GET("/patient-categories", h.PatientCategoriesList)
	r.GET("/patient-category/:id", h.PatientCategoryDetail)
	r.POST("/glomerular-calculation/:id/delete", h.DeleteGlomerularCalculation)
	r.POST("/patient-category/:id/add", h.CreateDraftGlomerularCalculationAndAddCategory)
	r.GET("/glomerular-calculation/:id", h.GlomerularCalculationDetail)
	r.GET("/images/:key", h.GetImage)

	api := r.Group("/api")
	{
		api.GET("/patient-categories", h.GetPatientCategoriesAPI)
		api.GET("/patient-categories/:id", h.GetPatientCategoryAPI)
		api.POST("/patient-categories", h.CreatePatientCategoryAPI)
		api.PUT("/patient-categories/:id", h.UpdatePatientCategoryAPI)
		api.DELETE("/patient-categories/:id", h.DeletePatientCategoryAPI)
		api.POST("/patient-categories/:id/add-to-calculation", h.AddCategoryToCalculationAPI)
		api.POST("/patient-categories/:id/image", h.UploadPatientCategoryImageAPI)

		api.GET("/cart-icon", h.GetCartIconAPI)
		api.GET("/glomerular-calculations", h.GetGlomerularCalculationsAPI)
		api.GET("/glomerular-calculations/:id", h.GetGlomerularCalculationAPI)
		api.PUT("/glomerular-calculations/:id", h.UpdateGlomerularCalculationAPI)
		api.PUT("/glomerular-calculations/:id/form", h.FormGlomerularCalculationAPI)
		api.PUT("/glomerular-calculations/:id/complete", h.CompleteGlomerularCalculationAPI)
		api.DELETE("/glomerular-calculations/:id", h.DeleteGlomerularCalculationAPI)

		api.PUT("/glomerular-calculations/:id/categories/:categoryId", h.UpdateGlomerularCalculationCategoryAPI)
		api.DELETE("/glomerular-calculations/:id/categories/:categoryId", h.DeleteGlomerularCalculationCategoryAPI)

		api.POST("/users/register", h.RegisterUserAPI)
		api.GET("/users/me", h.GetUserMeAPI)
		api.PUT("/users/me", h.UpdateUserMeAPI)
		api.POST("/users/auth", h.AuthUserAPI)
		api.POST("/users/logout", h.LogoutUserAPI)
	}

	r.NoRoute(h.Handle404)

	r.Run()
	log.Println("Server down")
}
