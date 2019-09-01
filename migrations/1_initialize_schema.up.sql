create table users (id serial unique, email text);


create table documents (id serial, author_id integer references users (id), text text);
