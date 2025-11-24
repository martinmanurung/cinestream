package users

import "time"

type User struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ExtID     string    `json:"ext_id" gorm:"ext_id;unique"`
	Name      string    `json:"name" gorm:"name"`
	Email     string    `json:"email" gorm:"email;unique"`
	Password  string    `json:"password" gorm:"password"`
	Role      string    `json:"role" gorm:"role"`
	CreatedAt time.Time `json:"created_at" gorm:"created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"updated_at"`
}

type UserRefreshToken struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserExtID string    `json:"user_ext_id" gorm:"column:user_ext_id;not null;index"`
	TokenHash string    `json:"token_hash" gorm:"token_hash;unique"`
	ExpiresAt time.Time `json:"expires_at" gorm:"expires_at"`
	CreatedAt time.Time `json:"created_at" gorm:"created_at"`
}

type UserRegisterRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type UserLoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	User         UserProfile `json:"user"`
}

type UserProfile struct {
	ExtID string `json:"ext_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UserRegisterResponse struct {
	ExtID string `json:"ext_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
