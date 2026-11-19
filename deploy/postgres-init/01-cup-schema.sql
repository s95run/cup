-- Local dev: simulate shared Postgres on vm-data with isolated schema cup.

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'cup_app') THEN
        CREATE ROLE cup_app WITH LOGIN PASSWORD 'cup_app_dev';
    END IF;
END
$$;

GRANT CONNECT ON DATABASE cup_dev TO cup_app;
CREATE SCHEMA IF NOT EXISTS cup AUTHORIZATION cup_app;
ALTER ROLE cup_app IN DATABASE cup_dev SET search_path TO cup;
GRANT USAGE, CREATE ON SCHEMA cup TO cup_app;
ALTER ROLE cup_app SET search_path TO cup;
REVOKE ALL ON SCHEMA public FROM cup_app;