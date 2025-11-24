-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    movie_id BIGINT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL COMMENT 'Harga film saat pembelian',
    payment_status ENUM('PENDING', 'PAID', 'FAILED', 'EXPIRED') NOT NULL DEFAULT 'PENDING',
    
    -- Kolom untuk Payment Gateway
    payment_gateway_ref VARCHAR(255) UNIQUE COMMENT 'ID order dari Midtrans/Xendit',
    checkout_url TEXT COMMENT 'Link redirect pembayaran untuk user',
    
    paid_at TIMESTAMP NULL COMMENT 'Diisi oleh webhook saat lunas',
    expires_at TIMESTAMP NULL COMMENT 'Waktu kedaluwarsa link checkout',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE RESTRICT
) ENGINE=InnoDB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS orders;
-- +goose StatementEnd
