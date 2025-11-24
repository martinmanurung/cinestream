package movies

import "time"

// Movie represents a movie entity in the database
type Movie struct {
	ID              int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Title           string    `json:"title" gorm:"type:varchar(255);not null"`
	Description     string    `json:"description" gorm:"type:text"`
	ReleaseDate     time.Time `json:"release_date" gorm:"type:date"`
	Director        string    `json:"director" gorm:"type:varchar(255)"`
	PosterURL       string    `json:"poster_url" gorm:"type:varchar(255)"`
	TrailerURL      string    `json:"trailer_url" gorm:"type:varchar(255)"`
	DurationMinutes int       `json:"duration_minutes"`
	Price           float64   `json:"price" gorm:"type:decimal(10,2);not null;default:0.00"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// MovieVideo represents the video processing status for a movie
type MovieVideo struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	MovieID        int64      `json:"movie_id" gorm:"uniqueIndex;not null"`
	UploadStatus   string     `json:"upload_status" gorm:"type:enum('PENDING','PROCESSING','READY','FAILED');default:'PENDING'"`
	RawFilePath    string     `json:"raw_file_path" gorm:"type:varchar(255)"`
	HLSPlaylistURL string     `json:"hls_playlist_url" gorm:"type:varchar(255)"`
	ErrorMessage   string     `json:"error_message" gorm:"type:text"`
	UploadedAt     time.Time  `json:"uploaded_at" gorm:"autoCreateTime"`
	ProcessedAt    *time.Time `json:"processed_at"`
}

// TableName overrides the table name for Movie
func (Movie) TableName() string {
	return "movies"
}

// TableName overrides the table name for MovieVideo
func (MovieVideo) TableName() string {
	return "movie_videos"
}

// Genre represents a movie genre
type Genre struct {
	ID   int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
}

// TableName overrides the table name for Genre
func (Genre) TableName() string {
	return "genres"
}

// MovieGenre represents the many-to-many relationship between movies and genres
type MovieGenre struct {
	MovieID int64 `json:"movie_id" gorm:"primaryKey;not null"`
	GenreID int   `json:"genre_id" gorm:"primaryKey;not null"`
}

// TableName overrides the table name for MovieGenre
func (MovieGenre) TableName() string {
	return "movie_genres"
}

// Request DTOs

// UploadMovieRequest represents the request to upload a new movie
type UploadMovieRequest struct {
	Title           string  `form:"title" validate:"required,min=1,max=255"`
	Description     string  `form:"description"`
	ReleaseDate     string  `form:"release_date"` // Format: YYYY-MM-DD
	Director        string  `form:"director" validate:"max=255"`
	PosterURL       string  `form:"poster_url" validate:"omitempty,url"`
	TrailerURL      string  `form:"trailer_url" validate:"omitempty,url"`
	DurationMinutes int     `form:"duration_minutes" validate:"omitempty,min=1"`
	Price           float64 `form:"price" validate:"required,min=0"`
	GenreIDs        []int   `form:"genre_ids"` // Optional: comma-separated genre IDs
}

// UpdateMovieRequest represents the request to update movie metadata
type UpdateMovieRequest struct {
	Title           string  `json:"title" validate:"omitempty,min=1,max=255"`
	Description     string  `json:"description"`
	ReleaseDate     string  `json:"release_date"` // Format: YYYY-MM-DD
	Director        string  `json:"director" validate:"omitempty,max=255"`
	PosterURL       string  `json:"poster_url" validate:"omitempty,url"`
	TrailerURL      string  `json:"trailer_url" validate:"omitempty,url"`
	DurationMinutes int     `json:"duration_minutes" validate:"omitempty,min=1"`
	Price           float64 `json:"price" validate:"omitempty,min=0"`
	GenreIDs        []int   `json:"genre_ids"` // Optional: update movie genres
}

// Response DTOs

// MovieListResponse represents a movie in the list view (catalog)
type MovieListResponse struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	PosterURL       string  `json:"poster_url"`
	Price           float64 `json:"price"`
	DurationMinutes int     `json:"duration_minutes"`
	UploadStatus    string  `json:"upload_status"`
}

// MovieDetailResponse represents detailed movie information
type MovieDetailResponse struct {
	ID              int64     `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	ReleaseDate     string    `json:"release_date"`
	Director        string    `json:"director"`
	PosterURL       string    `json:"poster_url"`
	TrailerURL      string    `json:"trailer_url"`
	DurationMinutes int       `json:"duration_minutes"`
	Price           float64   `json:"price"`
	UploadStatus    string    `json:"upload_status"`
	Genres          []string  `json:"genres,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UploadMovieResponse represents the response after uploading a movie
type UploadMovieResponse struct {
	MovieID int64  `json:"movie_id"`
	Message string `json:"message"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
	Limit       int   `json:"limit"`
}

// MovieListWithPagination represents paginated movie list
type MovieListWithPagination struct {
	Movies     []MovieListResponse `json:"movies"`
	Pagination PaginationMeta      `json:"pagination"`
}

// GenreRequest represents request to create a new genre
type GenreRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// GenreListResponse represents list of all genres
type GenreListResponse struct {
	Genres []Genre `json:"genres"`
}
