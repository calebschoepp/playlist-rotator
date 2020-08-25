CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  spotify_id      TEXT NOT NULL,
  playlists_built INTEGER NOT NULL,

  session_token  VARCHAR(128),
  session_expiry TIMESTAMPTZ,

  access_token  TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  token_type    TEXT NOT NULL,
  token_expiry  TIMESTAMPTZ NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE playlists (
  id      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID REFERENCES users ON DELETE RESTRICT,
  
  input       JSONB NOT NULL,
  name        TEXT NOT NULL,
  description TEXT NOT NULL,
  public      BOOLEAN NOT NULL,
  schedule    VARCHAR(64) NOT NULL,
  spotify_id  TEXT,
  failure_msg TEXT,
  building BOOLEAN NOT NULL DEFAULT FALSE,

  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_built_at TIMESTAMPTZ
);

CREATE OR REPLACE FUNCTION update_modified_column()
  RETURNS TRIGGER AS
$func$
BEGIN
  NEW.updated_at = now();
  return NEW;
END;
$func$ LANGUAGE 'plpgsql';

CREATE TRIGGER update_time_users
  BEFORE UPDATE
  ON users
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TRIGGER update_time_playlists
  BEFORE UPDATE
  ON playlists
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();