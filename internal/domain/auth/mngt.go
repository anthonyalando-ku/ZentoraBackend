package auth

import "time"

type AdminUserCard struct {
	IdentityID  int64    `json:"identity_id"`
	Email       string   `json:"email"`
	FullName    string   `json:"full_name"`
	AvatarURL   string   `json:"avatar_url"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
	LastLogin   *time.Time `json:"last_login,omitempty"`
}

type AdminUsersListResponse struct {
	Items []AdminUserCard `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

type AdminUsersStatsResponse struct {
	TotalUsers      int64 `json:"total_users"`
	ActiveUsers     int64 `json:"active_users"`
	InactiveUsers   int64 `json:"inactive_users"`
	SuspendedUsers  int64 `json:"suspended_users"`
	PendingUsers    int64 `json:"pending_verification_users"`
	AdminUsers      int64 `json:"admin_users"`
	SuperAdminUsers int64 `json:"super_admin_users"`
}

type AdminUsersListQuery struct {
	Page int `form:"page"`
	Size int `form:"page_size"`
}

type AdminUsersSearchQuery struct {
	Query string `form:"q"`
	Limit int    `form:"limit"`
}

type AdminDeleteUserRequest struct {
	Reason string `json:"reason"` // optional: for audit logs
}