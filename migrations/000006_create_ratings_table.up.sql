CREATE table if not exists ratings (
                                       product_id int REFERENCES products(product_id) ,
                                       user_id int references categories(id),
                                       rating int NOT NULL,
                                       comment text
)