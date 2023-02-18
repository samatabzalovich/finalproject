
CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       first_name VARCHAR(255) NOT NULL,
                       last_name VARCHAR(255) NOT NULL,
                       phone_number VARCHAR(255) NOT NULL,
                       email VARCHAR(255) NOT NULL,
                       password_hash bytea NOT NULL,
                       address VARCHAR(255),
                       activated bool,
                       profile_pic text NOT NULL  DEFAULT  'https://res.cloudinary.com/practicaldev/image/fetch/s--P1NWBtsb--/c_imagga_scale,f_auto,fl_progressive,h_900,q_auto,w_1600/https://thepracticaldev.s3.amazonaws.com/i/rlyibpr58qk49ci8y1rk.png',
                       version integer NOT NULL DEFAULT 1
);
CREATE TABLE if not exists products(
                                       product_id SERIAL PRIMARY KEY,
                                       created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
                                       title VARCHAR(255) NOT NULL,
                                       owner int REFERENCES users(id) NOT NULL,
                                       description text NOT NULL ,
                                       images text[],
                                       colors varchar(20)[],
                                       quantity integer NOT NULL default 1,
                                       price NUMERIC(10, 2),
                                        version integer NOT NULL DEFAULT 1

);