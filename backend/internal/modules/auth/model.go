package auth

import "time"

const (
	RoleAdmin     = "admin"
	RoleITManager = "it_manager"
	RoleEmployee  = "employee"

	StatusActive = "active"
	StatusLocked = "locked"
)

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	FullName       string    `json:"full_name"`
	EmployeeCode   string    `json:"employee_code"`
	DepartmentID   string    `json:"department_id,omitempty"`
	DepartmentName string    `json:"department_name,omitempty"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type RefreshSession struct {
	UserID    string
	TokenHash string
	ExpiresAt time.Time
}

type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresInSeconds int64  `json:"expires_in"`
	refreshExpiresAt time.Time
}

type AuthResult struct {
	Tokens TokenPair `json:"tokens"`
	User   User      `json:"user"`
}
