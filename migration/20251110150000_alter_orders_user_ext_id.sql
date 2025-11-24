-- +goose Up
-- Alter orders table: Change user_id to user_ext_id
ALTER TABLE orders 
  DROP FOREIGN KEY orders_ibfk_1,
  CHANGE COLUMN user_id user_ext_id VARCHAR(255) NOT NULL,
  ADD INDEX idx_orders_user_ext_id (user_ext_id);

-- Alter user_movie_access table: Change user_id to user_ext_id  
ALTER TABLE user_movie_access
  DROP FOREIGN KEY user_movie_access_ibfk_1,
  CHANGE COLUMN user_id user_ext_id VARCHAR(255) NOT NULL,
  ADD INDEX idx_user_movie_access_user_ext_id (user_ext_id);

-- +goose Down
-- Revert user_movie_access table
ALTER TABLE user_movie_access
  DROP INDEX idx_user_movie_access_user_ext_id,
  CHANGE COLUMN user_ext_id user_id BIGINT NOT NULL,
  ADD CONSTRAINT user_movie_access_ibfk_1 FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Revert orders table
ALTER TABLE orders
  DROP INDEX idx_orders_user_ext_id,
  CHANGE COLUMN user_ext_id user_id BIGINT NOT NULL,
  ADD CONSTRAINT orders_ibfk_1 FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
