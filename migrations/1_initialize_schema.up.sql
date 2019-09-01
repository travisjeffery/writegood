create table users (id serial unique, email text);


create table documents (id serial, author_id integer references users (id), text text);


create table sessions (key text primary key, user_id references (id), login_time timestamp,)