CREATE TABLE users (
  username TEXT NOT NULL PRIMARY KEY,
  displayname TEXT,
  salt BYTEA,
  password BYTEA
);

CREATE TABLE blogs (
  username TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  displayname TEXT,
  PRIMARY KEY (username, slug)
);

/* vim:set et ts=2 sw=2: */
