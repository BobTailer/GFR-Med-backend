package api

import (
	"context"
	"log"
	"os"
	"time"

	"ckd-epi/internal/app/auth"
	"ckd-epi/internal/app/database"
	"ckd-epi/internal/app/handler"
	"ckd-epi/internal/app/redis"
	"ckd-epi/internal/app/repository"
	"ckd-epi/internal/app/storage"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	_ "ckd-epi/docs" // Swagger документация
)

// @title CKD-EPI API
// @version 1.0
// @description API для расчета СКФ (скорость клубочковой фильтрации)

// @contact.name API Support

// @host localhost:8080
// @schemes http
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	ctx := context.Background()
	redisClient, err := redis.NewFromEnv(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	log.Println("Redis client initialized")

	jwtSecretKey := os.Getenv("JWT_SECRET_KEY")
	if jwtSecretKey == "" {
		jwtSecretKey = "default-secret-key-change-in-production"
		log.Println("Warning: Using default JWT secret key")
	}
	jwtService := auth.NewJWTService(jwtSecretKey)

	h := handler.NewHandler(repo, minioClient, jwtService, redisClient)
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

		apiAuth := api.Group("")
		apiAuth.Use(h.WithAuthCheck())
		{
			apiAuth.POST("/patient-categories", h.CreatePatientCategoryAPI)
			apiAuth.PUT("/patient-categories/:id", h.UpdatePatientCategoryAPI)
			apiAuth.DELETE("/patient-categories/:id", h.DeletePatientCategoryAPI)
			apiAuth.POST("/patient-categories/:id/add-to-calculation", h.AddCategoryToCalculationAPI)
			apiAuth.POST("/patient-categories/:id/image", h.UploadPatientCategoryImageAPI)

			apiAuth.GET("/cart-icon", h.GetCartIconAPI)
			apiAuth.GET("/glomerular-calculations", h.GetGlomerularCalculationsAPI)
			apiAuth.GET("/glomerular-calculations/:id", h.GetGlomerularCalculationAPI)
			apiAuth.PUT("/glomerular-calculations/:id", h.UpdateGlomerularCalculationAPI)
			apiAuth.PUT("/glomerular-calculations/:id/form", h.FormGlomerularCalculationAPI)
			apiAuth.PUT("/glomerular-calculations/:id/complete", h.CompleteGlomerularCalculationAPI)
			apiAuth.DELETE("/glomerular-calculations/:id", h.DeleteGlomerularCalculationAPI)

			apiAuth.PUT("/glomerular-calculations/:id/categories/:categoryId", h.UpdateGlomerularCalculationCategoryAPI)
			apiAuth.DELETE("/glomerular-calculations/:id/categories/:categoryId", h.DeleteGlomerularCalculationCategoryAPI)

			apiAuth.GET("/users/me", h.GetUserMeAPI)
			apiAuth.PUT("/users/me", h.UpdateUserMeAPI)
			apiAuth.POST("/users/logout", h.LogoutUserAPI)
		}

		api.POST("/users/register", h.RegisterUserAPI)
		api.POST("/users/auth", h.AuthUserAPI)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.NoRoute(h.Handle404)

	r.Run()
	log.Println("Server down")
}
