CREATE TABLE if not exists product_category (
                                                product_id int REFERENCES products(product_id) ON delete cascade ,
                                                category_id int references categories(id)
)