CREATE TABLE rentals (
    id BIGSERIAL PRIMARY KEY,
    card_id BIGINT NOT NULL REFERENCES cards(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    rented_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    status TEXT DEFAULT 'active'
);