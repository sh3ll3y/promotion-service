CREATE TABLE IF NOT EXISTS promotions (
                                          id UUID PRIMARY KEY,
                                          price DECIMAL(10, 2) NOT NULL,
    expiration_date TIMESTAMP NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_promotions_expiration_date ON promotions(expiration_date);