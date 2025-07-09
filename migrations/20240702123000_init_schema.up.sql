-- Users table
CREATE TABLE users (
                       id BIGSERIAL PRIMARY KEY,
                       name VARCHAR(100) NOT NULL,
                       phone_number VARCHAR(20) NOT NULL,
                       card_number VARCHAR(20) NOT NULL UNIQUE,
                       amount_of_money BIGINT NOT NULL, -- храним в тийинах
                       email VARCHAR(255) NOT NULL UNIQUE,
                       encrypted_password VARCHAR(255) NOT NULL,
                       is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);

-- Transactions table
CREATE TABLE transactions (
                              id BIGSERIAL PRIMARY KEY,
                              from_user_id BIGINT NOT NULL,
                              to_user_id BIGINT NOT NULL,
                              amount_of_money BIGINT NOT NULL CHECK (amount_of_money > 0), -- в тийинах
                              transaction_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                              is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                              FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
                              FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS users_phone_number_key ON users (phone_number);
CREATE UNIQUE INDEX IF NOT EXISTS users_card_number_key ON users (card_number);