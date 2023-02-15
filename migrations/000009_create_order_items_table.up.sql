CREATE TABLE if not exists order_items (
                             id SERIAL PRIMARY KEY,
                             order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
                             product_id INTEGER NOT NULL REFERENCES products(product_id) ON DELETE CASCADE,
                             quantity INTEGER NOT NULL
);