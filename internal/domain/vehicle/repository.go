// internal/domain/vehicle/repository.go
package vehicle

import "context"

type Repository interface {
	// Vehicle CRUD
	Create(ctx context.Context, vehicle *Vehicle) error
	FindByID(ctx context.Context, id int64) (*Vehicle, error)
	Update(ctx context.Context, id int64, vehicle *Vehicle) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filters *VehicleListFilters) ([]VehicleInfo, int64, error)
	
	// Utility
	ExistsByNumberPlate(ctx context.Context, numberPlate string) (bool, error)
	ExistsByLogbookNo(ctx context.Context, logbookNo string) (bool, error)
	
	// Vehicle Images
	AddImage(ctx context.Context, image *VehicleImage) error
	GetImages(ctx context.Context, vehicleID int64) ([]VehicleImage, error)
	DeleteImage(ctx context.Context, id int64) error
	SetPrimaryImage(ctx context.Context, vehicleID int64, imageID int64) error
	
	// Insurance
	CreateInsurance(ctx context.Context, insurance *VehicleInsurance) error
	GetInsurance(ctx context.Context, vehicleID int64) (*VehicleInsurance, error)
	GetActiveInsurance(ctx context.Context, vehicleID int64) (*VehicleInsurance, error)
	UpdateInsuranceStatus(ctx context.Context, id int64, status VerificationStatus) error
	
	// Inspection
	CreateInspection(ctx context.Context, inspection *VehicleInspection) error
	GetInspections(ctx context.Context, vehicleID int64) ([]VehicleInspection, error)
	GetLatestInspection(ctx context.Context, vehicleID int64) (*VehicleInspection, error)
	UpdateInspectionStatus(ctx context.Context, id int64, status VerificationStatus) error
	
	// Driver Assignment
	AssignDriverToVehicle(ctx context.Context, vehicleID, driverID int64) error
	//UnassignFromDriver(ctx context.Context, driverID int64, vehicleID int64) error
	GetVehiclesByDriver(ctx context.Context, driverID int64) ([]Vehicle, error)
	
	CreateCategory(ctx context.Context, c *Category) (*Category, error)
	UpdateCategory(ctx context.Context, c *Category) error
	DeleteCategory(ctx context.Context, id int64) error
	SetCategoryActiveStatus(ctx context.Context, id int64, isActive bool) error
	GetCategoryByID(ctx context.Context, id int64) (*Category, error)
	ListCategories(ctx context.Context) ([]Category, error)
	AddVehicleToCategory(ctx context.Context, categoryID, vehicleID int64) error
	RemoveVehicleFromCategory(ctx context.Context, categoryID, vehicleID int64) error
	ListCategoryVehicles(ctx context.Context, categoryID int64) ([]Vehicle, error)
}