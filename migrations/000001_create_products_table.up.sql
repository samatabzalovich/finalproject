CREATE TABLE if not exists products(
                                       product_id SERIAL PRIMARY KEY,
                                       title VARCHAR(255) NOT NULL,
                                       owner int REFERENCES users(id) NOT NULL,
                                       description text NOT NULL ,
                                       images text[],
                                       colors varchar(20)[],
                                       quantity integer NOT NULL default 1,
                                       price NUMERIC(10, 2)
);