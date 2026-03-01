package vehicle
// internal/domain/vehicle/entity.go
import "time"

type FuelType string
type TransmissionType string
type VerificationStatus string

const (
	FuelTypePetrol   FuelType = "petrol"
	FuelTypeDiesel   FuelType = "diesel"
	FuelTypeElectric FuelType = "electric"
	FuelTypeHybrid   FuelType = "hybrid"

	TransmissionManual    TransmissionType = "manual"
	TransmissionAutomatic TransmissionType = "automatic"

	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

// Vehicle represents a vehicle in the system
type Vehicle struct {
	ID               int64            `json:"id" db:"id"`
	CategoryID       int64            `json:"category_id" db:"category_id"`
	Name             string           `json:"name" db:"name"`
	Description      *string          `json:"description,omitempty" db:"description"`
	Type             string           `json:"type" db:"type"`
	SeatingCapacity  int              `json:"seating_capacity" db:"seating_capacity"`
	LuggageCapacity  *int             `json:"luggage_capacity,omitempty" db:"luggage_capacity"`
	HasWifi          bool             `json:"has_wifi" db:"has_wifi"`
	HasAC            bool             `json:"has_ac" db:"has_ac"`
	HasHeatedSeats   bool             `json:"has_heated_seats" db:"has_heated_seats"`
	HasGPS           bool             `json:"has_gps" db:"has_gps"`
	HasMusicSystem   bool             `json:"has_music_system" db:"has_music_system"`
	HasChargingPorts bool             `json:"has_charging_ports" db:"has_charging_ports"`
	CoverImage       *string          `json:"cover_image,omitempty" db:"cover_image"`
	Images           []string         `json:"images" db:"images"` // likely JSONB or TEXT[]
	IsAvailable      bool             `json:"is_available" db:"is_available"`
	DriverID         *int64           `json:"driver_id,omitempty" db:"driver_id"`
	NumberPlate      string           `json:"number_plate" db:"number_plate"`
	Model            string           `json:"model" db:"model"`
	YearMake         int              `json:"year_make" db:"year_make"`
	Color            string           `json:"color" db:"color"`
	EngineCapacity   string           `json:"engine_capacity" db:"engine_capacity"`
	Make             string           `json:"make" db:"make"`
	FuelType         FuelType         `json:"fuel_type" db:"fuel_type"`
	Transmission     TransmissionType `json:"transmission" db:"transmission"`
	IsActive         bool             `json:"is_active" db:"is_active"`
	CreatedBy        int64            `json:"created_by" db:"created_by"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
}


// VehicleImage represents vehicle images
type VehicleImage struct {
	ID           int64     `json:"id" db:"id"`
	VehicleID    int64     `json:"vehicle_id" db:"vehicle_id"`
	ImageURL     string    `json:"image_url" db:"image_url"`
	IsPrimary    bool      `json:"is_primary" db:"is_primary"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// VehicleInsurance represents vehicle insurance information
type VehicleInsurance struct {
	ID                 int64              `json:"id" db:"id"`
	VehicleID          int64              `json:"vehicle_id" db:"vehicle_id"`
	InsuranceProvider  string             `json:"insurance_provider" db:"insurance_provider"`
	PolicyNumber       string             `json:"policy_number" db:"policy_number"`
	InsuranceDocument  string             `json:"insurance_document" db:"insurance_document"`
	IssueDate          time.Time          `json:"issue_date" db:"issue_date"`
	ExpiryDate         time.Time          `json:"expiry_date" db:"expiry_date"`
	VerificationStatus VerificationStatus `json:"verification_status" db:"verification_status"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
}

// VehicleInspection represents vehicle inspection records
type VehicleInspection struct {
	ID                 int64              `json:"id" db:"id"`
	VehicleID          int64              `json:"vehicle_id" db:"vehicle_id"`
	InspectionDocument string             `json:"inspection_document" db:"inspection_document"`
	InspectionDate     time.Time          `json:"inspection_date" db:"inspection_date"`
	NextInspectionDate time.Time          `json:"next_inspection_date" db:"next_inspection_date"`
	VerificationStatus VerificationStatus `json:"verification_status" db:"verification_status"`
	Notes              *string            `json:"notes" db:"notes"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
}

type VehicleCInfo struct {
	Vehicle   *Vehicle            `json:"vehicle"`
	Category  *Category    `json:"category,omitempty"`
	Images    []*VehicleImage     `json:"images,omitempty"`
	Insurance *VehicleInsurance  `json:"insurance,omitempty"`
	Inspection *VehicleInspection `json:"inspection,omitempty"`
}

// DriverVehicleAssignment represents driver-vehicle assignments
type DriverVehicleAssignment struct {
	ID           int64      `json:"id" db:"id"`
	DriverID     int64      `json:"driver_id" db:"driver_id"`
	VehicleID    int64      `json:"vehicle_id" db:"vehicle_id"`
	AssignedAt   time.Time  `json:"assigned_at" db:"assigned_at"`
	UnassignedAt *time.Time `json:"unassigned_at" db:"unassigned_at"`
	IsActive     bool       `json:"is_active" db:"is_active"`
}

// CreateVehicleRequest for creating a new vehicle
type CreateVehicleRequest struct {
	CategoryID      int64            `json:"category_id" binding:"required"`
	NumberPlate     string           `json:"number_plate" binding:"required"`
	LogbookNo       string           `json:"logbook_no" binding:"required"`
	Color           string           `json:"color" binding:"required"`
	YearMake        int              `json:"year_make" binding:"required,min=1900,max=2100"`
	EngineCapacity  string           `json:"engine_capacity" binding:"required"`
	Make            string           `json:"make" binding:"required"`
	Model           string           `json:"model" binding:"required"`
	SeatingCapacity int              `json:"seating_capacity" binding:"required,min=1"`
	FuelType        FuelType         `json:"fuel_type" binding:"required"`
	Transmission    TransmissionType `json:"transmission" binding:"required"`
}

// UpdateVehicleRequest for updating vehicle details
type UpdateVehicleRequest struct {
	CategoryID      *int64            `json:"category_id"`
	NumberPlate     *string           `json:"number_plate"`
	Color           *string           `json:"color"`
	YearMake        *int              `json:"year_make" binding:"omitempty,min=1900,max=2100"`
	EngineCapacity  *string           `json:"engine_capacity"`
	Make            *string           `json:"make"`
	Model           *string           `json:"model"`
	SeatingCapacity *int              `json:"seating_capacity" binding:"omitempty,min=1"`
	FuelType        *FuelType         `json:"fuel_type"`
	Transmission    *TransmissionType `json:"transmission"`
	IsActive        *bool             `json:"is_active"`
}

// VehicleListFilters for listing/searching vehicles
type VehicleListFilters struct {
	CategoryID *int64  `form:"category_id"`
	IsActive   *bool   `form:"is_active"`
	FuelType   *FuelType `form:"fuel_type"`
	Search     string  `form:"search"` // Search by plate, make, model
	Page       int     `form:"page" binding:"min=1"`
	PageSize   int     `form:"page_size" binding:"min=1,max=100"`
	SortBy     string  `form:"sort_by"`
	SortOrder  string  `form:"sort_order" binding:"oneof=asc desc"`
}

// VehicleListResponse paginated list response
type VehicleListResponse struct {
	Vehicles   []VehicleInfo `json:"vehicles"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

// VehicleInfo represents vehicle information with related data
type VehicleInfo struct {
	ID              int64            `json:"id"`
	CategoryID      int64            `json:"category_id"`
	CategoryName    string           `json:"category_name"`
	NumberPlate     string           `json:"number_plate"`
	LogbookNo       string           `json:"logbook_no"`
	Color           string           `json:"color"`
	YearMake        int              `json:"year_make"`
	EngineCapacity  string           `json:"engine_capacity"`
	Make            string           `json:"make"`
	Model           string           `json:"model"`
	SeatingCapacity int              `json:"seating_capacity"`
	FuelType        FuelType         `json:"fuel_type"`
	Transmission    TransmissionType `json:"transmission"`
	IsActive        bool             `json:"is_active"`
	PrimaryImage    *string          `json:"primary_image"`
	CreatedAt       time.Time        `json:"created_at"`
	LuggageCapacity  *int             `json:"luggage_capacity,omitempty" db:"luggage_capacity"`

	Name             string           `json:"name" db:"name"`
	Description      *string          `json:"description,omitempty" db:"description"`
	Type             string           `json:"type" db:"type"`
	HasWifi          bool             `json:"has_wifi" db:"has_wifi"`
	HasAC            bool             `json:"has_ac" db:"has_ac"`
	HasHeatedSeats   bool             `json:"has_heated_seats" db:"has_heated_seats"`
	HasGPS           bool             `json:"has_gps" db:"has_gps"`
	HasMusicSystem   bool             `json:"has_music_system" db:"has_music_system"`
	HasChargingPorts bool             `json:"has_charging_ports" db:"has_charging_ports"`
	CoverImage       *string          `json:"cover_image,omitempty" db:"cover_image"`
	Images           []string         `json:"images" db:"images"` // likely JSONB or TEXT[]
	IsAvailable      bool             `json:"is_available" db:"is_available"`
	DriverID         *int64           `json:"driver_id,omitempty" db:"driver_id"`
	UpdatedAt 		time.Time        `json:"updated_at" db:"updated_at"`
}

// CreateInsuranceRequest for adding insurance
type CreateInsuranceRequest struct {
	InsuranceProvider string    `json:"insurance_provider" binding:"required"`
	PolicyNumber      string    `json:"policy_number" binding:"required"`
	InsuranceDocument string    `json:"insurance_document" binding:"required"`
	IssueDate         time.Time `json:"issue_date" binding:"required"`
	ExpiryDate        time.Time `json:"expiry_date" binding:"required"`
}

// CreateInspectionRequest for adding inspection
type CreateInspectionRequest struct {
	InspectionDocument string    `json:"inspection_document" binding:"required"`
	InspectionDate     time.Time `json:"inspection_date" binding:"required"`
	NextInspectionDate time.Time `json:"next_inspection_date" binding:"required"`
	Notes              *string   `json:"notes"`
}

// AssignVehicleRequest for assigning vehicle to driver
type AssignVehicleRequest struct {
	DriverID  int64 `json:"driver_id" binding:"required"`
	VehicleID int64 `json:"vehicle_id" binding:"required"`
}

// AddVehicleImageRequest for adding vehicle images
type AddVehicleImageRequest struct {
	ImageURL     string `json:"image_url" binding:"required"`
	IsPrimary    bool   `json:"is_primary"`
	DisplayOrder int    `json:"display_order"`
}