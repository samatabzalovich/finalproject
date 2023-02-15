CREATE TABLE if not exists favourites (
                                          product_id int REFERENCES products(product_id) ,
                                          user_id int references users(id)
);