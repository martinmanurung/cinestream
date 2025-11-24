-- +goose Up
-- Alter user_refresh_tokens table: Change user_ext_id from BIGINT to VARCHAR(255)
ALTER TABLE user_refresh_tokens
  DROP FOREIGN KEY user_refresh_tokens_ibfk_1,
  DROP INDEX idx_user_ext_id,
  MODIFY COLUMN user_ext_id VARCHAR(255) NOT NULL,
  ADD INDEX idx_user_ext_id (user_ext_id);

-- +goose Down
-- Revert user_refresh_tokens table: Change user_ext_id back to BIGINT
ALTER TABLE user_refresh_tokens
  DROP INDEX idx_user_ext_id,
  MODIFY COLUMN user_ext_id BIGINT NOT NULL,
  ADD INDEX idx_user_ext_id (user_ext_id),
  ADD CONSTRAINT user_refresh_tokens_ibfk_1 FOREIGN KEY (user_ext_id) REFERENCES users(id) ON DELETE CASCADE;
