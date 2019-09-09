create table users (
  id serial unique,
  email text unique,
  created timestamp with time zone default current_timestamp,
  updated timestamp with time zone default current_timestamp,
  signed_in timestamp with time zone
);
create table documents (
  id serial unique,
  author_id integer references users (id),
  text text,
  created timestamp with time zone default current_timestamp,
  updated timestamp with time zone default current_timestamp
);
create table document_logs (
  id serial unique,
  document_id integer references documents (id),
  text text,
  type text,
  diffs_html text,
  diffs bytea,
  created timestamp with time zone default current_timestamp,
  updated timestamp with time zone default current_timestamp
);
