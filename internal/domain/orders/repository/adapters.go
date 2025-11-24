package repository

import (
	"context"

	movieRepo "github.com/martinmanurung/cinestream/internal/domain/movies/repository"
	userRepo "github.com/martinmanurung/cinestream/internal/domain/users/repository"
)

// MovieRepositoryAdapter adapts the movie repository to order usecase interface
type MovieRepositoryAdapter struct {
	repo *movieRepo.MovieRepository
}

// NewMovieRepositoryAdapter creates a new movie repository adapter
func NewMovieRepositoryAdapter(repo *movieRepo.MovieRepository) *MovieRepositoryAdapter {
	return &MovieRepositoryAdapter{repo: repo}
}

// FindMovieByID adapts the movie repository method
func (a *MovieRepositoryAdapter) FindMovieByID(movieID int64) (map[string]interface{}, error) {
	movie, err := (*a.repo).FindMovieByID(context.Background(), movieID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":    movie.ID,
		"title": movie.Title,
		"price": movie.Price,
	}, nil
}

// GetMovieHLSURL gets the HLS URL for a movie
func (a *MovieRepositoryAdapter) GetMovieHLSURL(movieID int64) (string, error) {
	return (*a.repo).GetHLSURL(context.Background(), movieID)
}

// UserRepositoryAdapter adapts the user repository to order usecase interface
type UserRepositoryAdapter struct {
	repo *userRepo.User
}

// NewUserRepositoryAdapter creates a new user repository adapter
func NewUserRepositoryAdapter(repo *userRepo.User) *UserRepositoryAdapter {
	return &UserRepositoryAdapter{repo: repo}
}

// FindUserByExtID adapts the user repository method to find user by external ID
func (a *UserRepositoryAdapter) FindUserByExtID(userExtID string) (map[string]interface{}, error) {
	user, err := (*a.repo).FindUserByExtID(context.Background(), userExtID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":     user.ID,
		"ext_id": user.ExtID,
		"name":   user.Name,
		"email":  user.Email,
		"role":   user.Role,
	}, nil
}
