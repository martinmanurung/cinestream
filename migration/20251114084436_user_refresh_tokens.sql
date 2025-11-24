-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_refresh_tokens (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_ext_id BIGINT NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_ext_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_token_hash (token_hash),
    INDEX idx_user_ext_id (user_ext_id),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_refresh_tokens;
-- +goose StatementEnd
