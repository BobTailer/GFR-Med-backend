package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"ckd-epi/internal/app/models"
	"ckd-epi/internal/app/storage"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() error {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")
	dbUser := getEnv("DB_USER", "ckd_user")
	dbPassword := getEnv("DB_PASSWORD", "ckd_password")
	dbName := getEnv("DB_NAME", "ckd_epi")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Moscow",
		dbHost, dbUser, dbPassword, dbName, dbPort)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")

	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func AutoMigrate() error {
	err := DB.AutoMigrate(
		&models.PatientCategory{},
		&models.User{},
		&models.GlomerularCalculation{},
		&models.GlomerularCalculationCategory{},
	)

	if err != nil {
		return err
	}

	log.Println("Database migrations completed")

	if err := DB.Exec("ALTER TABLE patient_categories DROP COLUMN IF EXISTS min_creatinine").Error; err != nil {
		log.Printf("Warning: could not drop min_creatinine column: %v", err)
	}
	if err := DB.Exec("ALTER TABLE patient_categories DROP COLUMN IF EXISTS max_creatinine").Error; err != nil {
		log.Printf("Warning: could not drop max_creatinine column: %v", err)
	}
	if err := DB.Exec("ALTER TABLE patient_categories DROP COLUMN IF EXISTS avatar").Error; err != nil {
		log.Printf("Warning: could not drop avatar column: %v", err)
	}

	if err := DB.Exec("ALTER TABLE glomerular_calculation_categories DROP COLUMN IF EXISTS quantity").Error; err != nil {
		log.Printf("Warning: could not drop quantity column: %v", err)
	}
	if err := DB.Exec("ALTER TABLE glomerular_calculation_categories DROP COLUMN IF EXISTS order_num").Error; err != nil {
		log.Printf("Warning: could not drop order_num column: %v", err)
	}
	if err := DB.Exec("ALTER TABLE glomerular_calculation_categories DROP COLUMN IF EXISTS is_main").Error; err != nil {
		log.Printf("Warning: could not drop is_main column: %v", err)
	}

	if DB.Migrator().HasTable("gfr_orders") {
		if err := DB.Exec("ALTER TABLE gfr_orders RENAME TO glomerular_calculations").Error; err != nil {
			log.Printf("Warning: could not rename gfr_orders table: %v", err)
		}
	}

	if DB.Migrator().HasTable("gfr_order_categories") {
		var hasServiceID bool
		DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'gfr_order_categories' AND column_name = 'service_id')").Scan(&hasServiceID)
		if hasServiceID {
			if err := DB.Exec("ALTER TABLE gfr_order_categories RENAME COLUMN service_id TO category_id").Error; err != nil {
				log.Printf("Warning: could not rename service_id to category_id: %v", err)
			}
		}
		var hasGFROrderID bool
		DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'gfr_order_categories' AND column_name = 'gfr_order_id')").Scan(&hasGFROrderID)
		if hasGFROrderID {
			if err := DB.Exec("ALTER TABLE gfr_order_categories RENAME COLUMN gfr_order_id TO order_id").Error; err != nil {
				log.Printf("Warning: could not rename gfr_order_id to order_id: %v", err)
			}
		}
		var hasOrderID bool
		DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'gfr_order_categories' AND column_name = 'order_id')").Scan(&hasOrderID)
		if hasOrderID {
			if err := DB.Exec("ALTER TABLE gfr_order_categories RENAME COLUMN order_id TO calculation_id").Error; err != nil {
				log.Printf("Warning: could not rename order_id to calculation_id: %v", err)
			}
		}
		if err := DB.Exec("ALTER TABLE gfr_order_categories RENAME TO glomerular_calculation_categories").Error; err != nil {
			log.Printf("Warning: could not rename gfr_order_categories table: %v", err)
		}
	}

	var hasDoctorName bool
	var hasAverageAge bool
	if DB.Migrator().HasTable("glomerular_calculations") {
		DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'glomerular_calculations' AND column_name = 'doctor_name')").Scan(&hasDoctorName)
		if !hasDoctorName {
			if err := DB.Exec("ALTER TABLE glomerular_calculations ADD COLUMN IF NOT EXISTS doctor_name VARCHAR(255)").Error; err != nil {
				log.Printf("Warning: could not add doctor_name column: %v", err)
			}
		}
		DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'glomerular_calculations' AND column_name = 'average_age')").Scan(&hasAverageAge)
		if !hasAverageAge {
			if err := DB.Exec("ALTER TABLE glomerular_calculations ADD COLUMN IF NOT EXISTS average_age DECIMAL(10,2)").Error; err != nil {
				log.Printf("Warning: could not add average_age column: %v", err)
			}
		}
	}
	var hasCalculatedGFR bool
	if DB.Migrator().HasTable("glomerular_calculation_categories") {
		DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'glomerular_calculation_categories' AND column_name = 'calculated_gfr')").Scan(&hasCalculatedGFR)
		if !hasCalculatedGFR {
			if err := DB.Exec("ALTER TABLE glomerular_calculation_categories ADD COLUMN IF NOT EXISTS calculated_gfr DECIMAL(10,2)").Error; err != nil {
				log.Printf("Warning: could not add calculated_gfr column: %v", err)
			}
		}
	}

	if !DB.Migrator().HasIndex(&models.GlomerularCalculationCategory{}, "calculation_id") {
		if err := DB.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_glomerular_calculation_category_unique 
			ON glomerular_calculation_categories(calculation_id, category_id);
		`).Error; err != nil {
			log.Printf("Warning: could not create unique index: %v", err)
		}
	}

	if err := DB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_glomerular_calculation_draft_unique 
		ON glomerular_calculations(creator_id) 
		WHERE status = 'черновик' AND deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: could not create draft unique index: %v", err)
	}

	return nil
}

func SeedData(minioClient *storage.MinIOClient) error {
	var count int64
	DB.Model(&models.PatientCategory{}).Count(&count)
	if count > 0 {
		log.Println("Database already seeded")
		return nil
	}

	user := models.User{
		Username:    "admin",
		Password:    "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		IsModerator: true,
	}
	if err := DB.Create(&user).Error; err != nil {
		log.Printf("Warning: could not create admin user: %v", err)
	}

	imageBasePath := getEnv("IMAGE_BASE_PATH", "./resources/img")

	categoryConfigs := []struct {
		name        string
		description string
		gender      string
		age         int
		avatarFile  string
	}{
		{"Мужчина, 30-40 лет", "Категория: мужской пол, возраст от 30 до 40 лет", "М", 35, "man_30_40.png"},
		{"Мужчина, 40-50 лет", "Категория: мужской пол, возраст от 40 до 50 лет", "М", 45, "man_40_50.png"},
		{"Мужчина, 50-60 лет", "Категория: мужской пол, возраст от 50 до 60 лет", "М", 55, "man_50_60.png"},
		{"Женщина, 40-50 лет", "Категория: женский пол, возраст от 40 до 50 лет", "Ж", 45, "woman_40_50.png"},
		{"Женщина, 50-60 лет", "Категория: женский пол, возраст от 50 до 60 лет", "Ж", 55, "woman_50_60.png"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, config := range categoryConfigs {
		category := models.PatientCategory{
			Name:        config.name,
			Description: config.description,
			Gender:      config.gender,
			Age:         config.age,
		}

		imageFile := config.avatarFile
		imagePath := filepath.Join(imageBasePath, imageFile)

		if _, err := os.Stat(imagePath); err == nil {
			imageKey := storage.GenerateObjectKey(imageFile)

			exists, err := minioClient.ObjectExists(ctx, imageKey)
			if err != nil {
				log.Printf("Warning: could not check if image exists: %v", err)
			}

			if !exists {
				contentType := "image/png"
				if err := minioClient.UploadFile(ctx, imageKey, imagePath, contentType); err != nil {
					log.Printf("Warning: could not upload image %s to MinIO: %v", imagePath, err)
				} else {
					log.Printf("Uploaded image to MinIO: %s -> %s", imagePath, imageKey)
				}
			}

			category.ImageKey = &imageKey
		} else {
			log.Printf("Warning: image file not found: %s", imagePath)
		}

		if err := DB.Create(&category).Error; err != nil {
			log.Printf("Warning: could not create category %s: %v", category.Name, err)
		}
	}

	log.Println("Database seeded successfully")
	return nil
}

func stringPtr(s string) *string {
	return &s
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
