package repository

type Patient struct {
	ID            int
	Name          string
	Gender        string
	Age           int
	MinCreatinine float64
	MaxCreatinine float64
	Avatar        string
}

type Order struct {
	ID      int
	Patient Patient
	Result  string
}

type Repository struct {
	SelectedPatientID int
}

func NewRepository() (*Repository, error) {
	return &Repository{SelectedPatientID: 1}, nil // по умолчанию первый пациент
}

func (r *Repository) GetPatients() []Patient {
	return []Patient{
		{ID: 1, Name: "Иванов Иван Иванович", Gender: "М", Age: 19, MinCreatinine: 88.4, MaxCreatinine: 104.6, Avatar: "male.png"},
		{ID: 2, Name: "Зайцев Егор Олегович", Gender: "М", Age: 22, MinCreatinine: 90.0, MaxCreatinine: 110.2, Avatar: "male.png"},
		{ID: 3, Name: "Рыбина Мария Александровна", Gender: "Ж", Age: 21, MinCreatinine: 70.3, MaxCreatinine: 99.8, Avatar: "female.png"},
		{ID: 4, Name: "Первая Лидия Егоровна", Gender: "Ж", Age: 24, MinCreatinine: 75.1, MaxCreatinine: 102.2, Avatar: "female.png"},
		{ID: 5, Name: "Панин Михаил Михайлович", Gender: "М", Age: 28, MinCreatinine: 80.0, MaxCreatinine: 115.3, Avatar: "male.png"},
		{ID: 6, Name: "Кошин Андрей Владимирович", Gender: "М", Age: 33, MinCreatinine: 93.5, MaxCreatinine: 120.0, Avatar: "male.png"},
		{ID: 7, Name: "Черная Валерия Артемовна", Gender: "Ж", Age: 26, MinCreatinine: 72.8, MaxCreatinine: 98.4, Avatar: "female.png"},
		{ID: 8, Name: "Черный Матвей Артемович", Gender: "М", Age: 20, MinCreatinine: 85.2, MaxCreatinine: 102.0, Avatar: "male.png"},
	}
}

func (r *Repository) GetPatientByID(id int) (Patient, bool) {
	for _, p := range r.GetPatients() {
		if p.ID == id {
			return p, true
		}
	}
	return Patient{}, false
}

// Для поиска
func (r *Repository) FilterPatients(query string) []Patient {
	patients := r.GetPatients()
	if query == "" {
		return patients
	}
	var result []Patient
	for _, p := range patients {
		if containsIgnoreCase(p.Name, query) {
			result = append(result, p)
		}
	}
	return result
}

func containsIgnoreCase(str, substr string) bool {
	return len(substr) == 0 || (len(str) > 0 && (caseInsensitiveContains(str, substr)))
}
func caseInsensitiveContains(str, substr string) bool {
	return len(substr) == 0 || (len(str) > 0 && (contains(str, substr)))
}
func contains(str, substr string) bool {
	return len(substr) == 0 || (len(str) > 0 && (stringContains(str, substr)))
}
func stringContains(str, substr string) bool {
	return len(substr) == 0 || (len(str) > 0 && (indexOf(str, substr) >= 0))
}
func indexOf(str, substr string) int {
	return len(str) - len(substr)
}

// Для расчёта СКФ — просто возвращаем фиксированный результат (можно заменить на формулу)
func (r *Repository) GetOrder(patientID int) (Order, bool) {
	patient, found := r.GetPatientByID(patientID)
	if !found {
		return Order{}, false
	}
	return Order{
		ID:      patientID,
		Patient: patient,
		Result:  "86 мл/мин/1.73м²", // здесь формируется итог
	}, true
}

// Для кнопки "Выбрать" — сохраняем выбранного пациента
func (r *Repository) SelectPatient(id int) {
	r.SelectedPatientID = id
}
