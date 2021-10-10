DROP TABLE IF EXISTS requests;

create table requests(
    id SERIAL PRIMARY KEY,
    host TEXT,
    path TEXT,
    method TEXT,
    headers jsonb,
    params TEXT,
    cookies jsonb,
    body text,
    schema text
);