CREATE table if not exists ratings (
                                       product_id int REFERENCES products(product_id) On delete cascade ,
                                       user_id int references categories(id) ON delete cascade ,
                                       rating int NOT NULL,
                                       comment text
)