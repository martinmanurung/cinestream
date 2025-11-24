-- +goose Up
-- +goose StatementBegin
CREATE TABLE movie_videos (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    movie_id BIGINT NOT NULL UNIQUE,
    upload_status ENUM('PENDING', 'PROCESSING', 'READY', 'FAILED') NOT NULL DEFAULT 'PENDING',
    raw_file_path VARCHAR(255) COMMENT 'Path MinIO bucket raw-videos',
    hls_playlist_url VARCHAR(255) COMMENT 'Path master.m3u8 di MinIO processed-videos',
    error_message TEXT COMMENT 'Jika status FAILED',
    uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL,
    
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
) ENGINE=InnoDB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS movie_videos;
-- +goose StatementEnd
