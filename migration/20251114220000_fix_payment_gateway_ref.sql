-- +goose Up
-- +goose StatementBegin
-- Set empty string payment_gateway_ref to NULL
UPDATE orders 
SET payment_gateway_ref = NULL 
WHERE payment_gateway_ref = '';
-- +goose StatementEnd

-- +goose StatementBegin
-- Ensure the column allows NULL and has UNIQUE constraint
ALTER TABLE orders 
MODIFY COLUMN payment_gateway_ref VARCHAR(255) NULL UNIQUE COMMENT 'ID order dari Midtrans/Xendit';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No need to revert, this is a data fix
-- +goose StatementEnd
