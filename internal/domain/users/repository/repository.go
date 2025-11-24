package repository

import (
	"context"
	"errors"

	"github.com/martinmanurung/cinestream/internal/domain/users"
	"gorm.io/gorm"
)

type User struct {
	db *gorm.DB
}

func NewUser(db *gorm.DB) *User {
	return &User{db: db}
}

func (u User) CreateNewUser(ctx context.Context, user users.User) error {
	if err := u.db.Create(&user).Error; err != nil {
		return err
	}
	return nil
}

func (u User) FindUserByEmail(ctx context.Context, email string) (*users.User, error) {
	var user users.User
	err := u.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		// Jika record tidak ditemukan, return nil tanpa error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u User) FindUserByExtID(ctx context.Context, extID string) (*users.User, error) {
	var user users.User
	err := u.db.Where("ext_id = ?", extID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u User) FindUserByID(ctx context.Context, userID int) (*users.User, error) {
	var user users.User
	err := u.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u User) CreateRefreshToken(ctx context.Context, token users.UserRefreshToken) error {
	return u.db.WithContext(ctx).Create(&token).Error
}

func (u User) FindRefreshToken(ctx context.Context, tokenHash string) (*users.UserRefreshToken, error) {
	var token users.UserRefreshToken
	err := u.db.WithContext(ctx).
		Where("token_hash = ? AND expires_at > NOW()", tokenHash).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (u User) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	return u.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		Delete(&users.UserRefreshToken{}).Error
}
