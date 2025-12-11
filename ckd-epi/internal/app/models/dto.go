package models

type PatientCategoryResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Gender      string  `json:"gender"`
	Age         int     `json:"age"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type PatientCategoryListResponse struct {
	Categories []PatientCategoryResponse `json:"categories"`
}

type GlomerularCalculationCategoryResponse struct {
	CategoryID      uint     `json:"category_id"`
	CreatinineLevel *float64 `json:"creatinine_level,omitempty"`
	CalculatedGFR   *float64 `json:"calculated_gfr,omitempty"`
	Category        PatientCategoryResponse `json:"category"`
}

type GlomerularCalculationResponse struct {
	ID                uint                                    `json:"id"`
	Status            CalculationStatus                       `json:"status"`
	CreatedAt         string                                  `json:"created_at"`
	CreatorID         uint                                    `json:"creator_id"`
	CreatorUsername   *string                                 `json:"creator_username,omitempty"`
	DoctorName        *string                                 `json:"doctor_name,omitempty"`
	AverageAge        *float64                                `json:"average_age,omitempty"`
	FormedAt          *string                                 `json:"formed_at,omitempty"`
	CompletedAt       *string                                 `json:"completed_at,omitempty"`
	ModeratorID       *uint                                   `json:"moderator_id,omitempty"`
	ModeratorUsername *string                                 `json:"moderator_username,omitempty"`
	CalculatedGFR     *float64                                `json:"calculated_gfr,omitempty"`
	Categories        []GlomerularCalculationCategoryResponse `json:"categories"`
}

type GlomerularCalculationListItem struct {
	ID                  uint           `json:"id"`
	Status              CalculationStatus `json:"status"`
	CreatedAt           string         `json:"created_at"`
	CreatorID           uint           `json:"creator_id"`
	CreatorUsername     string         `json:"creator_username"`
	DoctorName          *string        `json:"doctor_name,omitempty"`
	AverageAge          *float64       `json:"average_age,omitempty"`
	FormedAt            *string        `json:"formed_at,omitempty"`
	CompletedAt         *string        `json:"completed_at,omitempty"`
	ModeratorID         *uint          `json:"moderator_id,omitempty"`
	ModeratorUsername   *string        `json:"moderator_username,omitempty"`
	CalculatedGFR       *float64       `json:"calculated_gfr,omitempty"`
	CategoriesWithGFR   int            `json:"categories_with_gfr"`
}

type GlomerularCalculationListResponse struct {
	Calculations []GlomerularCalculationListItem `json:"calculations"`
}

type CartIconResponse struct {
	ID    *uint `json:"id,omitempty"`
	Count int   `json:"count"`
}

type UserResponse struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	IsModerator bool   `json:"is_moderator"`
	CreatedAt   string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type CreatePatientCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Gender      string `json:"gender" binding:"required"`
	Age         int    `json:"age" binding:"required"`
}

type UpdatePatientCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Gender      *string `json:"gender"`
	Age         *int    `json:"age"`
}

type UpdateGlomerularCalculationRequest struct {
	DoctorName *string `json:"doctor_name"`
}

type FormGlomerularCalculationRequest struct {
}

type CompleteGlomerularCalculationRequest struct {
	Action string `json:"action" binding:"required"`
}

type UpdateGlomerularCalculationCategoryRequest struct {
	CreatinineLevel *float64 `json:"creatinine_level"`
}

type RegisterUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateUserRequest struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  UserResponse `json:"user"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

