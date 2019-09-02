create table users (id serial unique, email text, created timestamp with time zone, updated timestamp with time zone);


create table documents (id serial, author_id integer references users (id), text text, created timestamp with time zone, updated timestamp with time zone);
