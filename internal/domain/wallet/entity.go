package wallet
// internal/domain/wallet/entity.go

import "time"

type TransactionType string
type EarningStatus string
type WithdrawalMethod string
type WithdrawalStatus string

const (
	TransactionRideEarning TransactionType = "ride_earning"
	TransactionWithdrawal  TransactionType = "withdrawal"
	TransactionBonus       TransactionType = "bonus"
	TransactionPenalty     TransactionType = "penalty"
	TransactionRefund      TransactionType = "refund"

	EarningPending   EarningStatus = "pending"
	EarningCleared   EarningStatus = "cleared"
	EarningLocked    EarningStatus = "locked"
	EarningWithdrawn EarningStatus = "withdrawn"

	WithdrawalMethodMpesa        WithdrawalMethod = "mpesa"
	WithdrawalMethodBankTransfer WithdrawalMethod = "bank_transfer"

	WithdrawalStatusPending    WithdrawalStatus = "pending"
	WithdrawalStatusProcessing WithdrawalStatus = "processing"
	WithdrawalStatusCompleted  WithdrawalStatus = "completed"
	WithdrawalStatusRejected   WithdrawalStatus = "rejected"
)

// DriverWallet represents driver wallet
type DriverWallet struct {
	ID            int64     `json:"id" db:"id"`
	DriverID      int64     `json:"driver_id" db:"driver_id"`
	Balance       float64   `json:"balance" db:"balance"`
	LockedBalance float64   `json:"locked_balance" db:"locked_balance"`
	LastUpdated   time.Time `json:"last_updated" db:"last_updated"`
}

// DriverEarning represents a transaction in driver's wallet
type DriverEarning struct {
	ID              int64           `json:"id" db:"id"`
	DriverID        int64           `json:"driver_id" db:"driver_id"`
	TripID          *int64          `json:"trip_id" db:"trip_id"`
	TransactionType TransactionType `json:"transaction_type" db:"transaction_type"`
	Amount          float64         `json:"amount" db:"amount"`
	Currency        string          `json:"currency" db:"currency"`
	BalanceBefore   float64         `json:"balance_before" db:"balance_before"`
	BalanceAfter    float64         `json:"balance_after" db:"balance_after"`
	Status          EarningStatus   `json:"status" db:"status"`
	ClearedAt       *time.Time      `json:"cleared_at" db:"cleared_at"`
	Description     *string         `json:"description" db:"description"`
	ReferenceNo     *string         `json:"reference_no" db:"reference_no"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// WithdrawalRequest represents a withdrawal request
type WithdrawalRequest struct {
	ID              int64            `json:"id" db:"id"`
	DriverID        int64            `json:"driver_id" db:"driver_id"`
	Amount          float64          `json:"amount" db:"amount"`
	Currency        string           `json:"currency" db:"currency"`
	WithdrawalMethod WithdrawalMethod `json:"withdrawal_method" db:"withdrawal_method"`
	AccountDetails  string           `json:"account_details" db:"account_details"`
	Status          WithdrawalStatus `json:"status" db:"status"`
	ProcessedAt     *time.Time       `json:"processed_at" db:"processed_at"`
	ProcessedBy     *int64           `json:"processed_by" db:"processed_by"`
	RejectionReason *string          `json:"rejection_reason" db:"rejection_reason"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

// DTOs for requests/responses

type WalletSummary struct {
	TotalBalance      float64 `json:"total_balance"`
	AvailableBalance  float64 `json:"available_balance"`
	LockedBalance     float64 `json:"locked_balance"`
	PendingClearance  float64 `json:"pending_clearance"`
	Currency          string  `json:"currency"`
}

type EarningsSummary struct {
	TotalEarnings  float64 `json:"total_earnings"`
	WeekEarnings   float64 `json:"week_earnings"`
	MonthEarnings  float64 `json:"month_earnings"`
	TodayEarnings  float64 `json:"today_earnings"`
	Currency       string  `json:"currency"`
}

type TransactionHistory struct {
	ID              int64           `json:"id"`
	TransactionType TransactionType `json:"transaction_type"`
	Amount          float64         `json:"amount"`
	Currency        string          `json:"currency"`
	Description     string          `json:"description"`
	Status          EarningStatus   `json:"status"`
	Date            time.Time       `json:"date"`
	TripID          *int64          `json:"trip_id,omitempty"`
}

type TransactionListFilters struct {
	TransactionType *TransactionType `form:"transaction_type"`
	Status          *EarningStatus   `form:"status"`
	StartDate       *time.Time       `form:"start_date"`
	EndDate         *time.Time       `form:"end_date"`
	Page            int              `form:"page" binding:"min=1"`
	PageSize        int              `form:"page_size" binding:"min=1,max=100"`
}

type CreateWithdrawalRequest struct {
	Amount           float64          `json:"amount" binding:"required,gt=0"`
	WithdrawalMethod WithdrawalMethod `json:"withdrawal_method" binding:"required"`
	AccountDetails   string           `json:"account_details" binding:"required"`
}

type WithdrawalListResponse struct {
	Withdrawals []WithdrawalRequest `json:"withdrawals"`
	Total       int64               `json:"total"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
	TotalPages  int                 `json:"total_pages"`
}