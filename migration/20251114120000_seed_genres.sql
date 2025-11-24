-- +goose Up
-- +goose StatementBegin
INSERT INTO genres (name) VALUES
    ('Action'),
    ('Adventure'),
    ('Animation'),
    ('Comedy'),
    ('Crime'),
    ('Documentary'),
    ('Drama'),
    ('Fantasy'),
    ('Horror'),
    ('Mystery'),
    ('Romance'),
    ('Science Fiction'),
    ('Thriller'),
    ('War'),
    ('Western');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM genres WHERE name IN (
    'Action', 'Adventure', 'Animation', 'Comedy', 'Crime', 
    'Documentary', 'Drama', 'Fantasy', 'Horror', 'Mystery', 
    'Romance', 'Science Fiction', 'Thriller', 'War', 'Western'
);
-- +goose StatementEnd
