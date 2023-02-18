ALTER TABLE products ADD CONSTRAINT quantity_check CHECK (quantity >= 0);
