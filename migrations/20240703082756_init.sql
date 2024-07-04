-- +goose Up
CREATE TABLE promotions (
                            id UUID PRIMARY KEY,
                            price DECIMAL(10, 2) NOT NULL,
                            expiration_date TIMESTAMP NOT NULL
);

CREATE INDEX idx_promotions_id ON promotions(id);

CREATE TABLE promotions_temp (
                                 id UUID PRIMARY KEY,
                                 price DECIMAL(10, 2) NOT NULL,
                                 expiration_date TIMESTAMP NOT NULL
);

CREATE INDEX idx_promotions_temp_id ON promotions_temp(id);

-- +goose Down
DROP TABLE IF EXISTS promotions;
DROP TABLE IF EXISTS promotions_temp;