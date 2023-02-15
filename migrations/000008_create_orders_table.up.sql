
CREATE TABLE if not exists orders (
                        id SERIAL PRIMARY KEY,
                        user_id int NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                        ordered_at TIMESTAMP NOT NULL,
                        status INTEGER DEFAULT 0,
                        total_price DECIMAL(10,2) NOT NULL,
                        address VARCHAR(255) NOT NULL
);