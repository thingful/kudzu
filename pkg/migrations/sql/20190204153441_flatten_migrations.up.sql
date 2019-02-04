CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE TABLE IF NOT EXISTS public.users (
  id          SERIAL PRIMARY KEY,
  uid         VARCHAR(250) UNIQUE NOT NULL CHECK (uid <> ''),
  created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT(NOW() AT TIME ZONE 'utc')
);

CREATE TABLE IF NOT EXISTS public.identities (
  id            SERIAL PRIMARY KEY,
  owner_id      INTEGER REFERENCES users(id) ON DELETE CASCADE,
  auth_provider VARCHAR(250),
  access_token  VARCHAR(250),
  refresh_token VARCHAR(250),
  created_at    TIMESTAMP WITHOUT TIME ZONE DEFAULT(NOW() AT TIME ZONE 'utc')
);

CREATE TABLE IF NOT EXISTS public.things (
  id                    SERIAL PRIMARY KEY,
  uid                   VARCHAR UNIQUE NOT NULL,
  owner_id              INTEGER REFERENCES users(id) ON DELETE SET NULL,
  data_url              VARCHAR UNIQUE,
  provider              VARCHAR(250),
  serial_num            VARCHAR(100),
  lat                   DOUBLE PRECISION NOT NULL,
  long                  DOUBLE PRECISION NOT NULL,
  first_sample          TIMESTAMP WITH TIME ZONE,
  last_sample           TIMESTAMP WITH TIME ZONE,
  created_at            TIMESTAMP WITH TIME ZONE DEFAULT (now() at time zone 'utc'),
  indexed_at            TIMESTAMP WITH TIME ZONE,
  resource_url          TEXT NOT NULL,
  updated_at            TIMESTAMP WITHOUT TIME ZONE,
  nickname              TEXT DEFAULT ''::text,
  last_uploaded_sample  TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS public.applications (
  id         SERIAL PRIMARY KEY,
  uid        VARCHAR(10) NOT NULL UNIQUE,
  app_name   VARCHAR(250) NOT NULL UNIQUE,
  created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() at time zone 'utc'),
  key_hash   VARCHAR(250) UNIQUE,
  scope      TEXT[]
);

CREATE TABLE IF NOT EXISTS public.data_sources (
  id        SERIAL PRIMARY KEY,
  name      VARCHAR(250) UNIQUE NOT NULL CHECK (name <> ''),
  unit      VARCHAR(250),
  data_type VARCHAR(250) NOT NULL
);

CREATE TABLE IF NOT EXISTS public.channels (
  id             SERIAL PRIMARY KEY,
  thing_uid      TEXT REFERENCES things(uid) ON DELETE CASCADE,
  data_source_id INTEGER REFERENCES data_sources(id) ON DELETE CASCADE,
  UNIQUE(thing_uid, data_source_id)
);