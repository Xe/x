CREATE TABLE IF NOT EXISTS articles
  ( id              TEXT  NOT NULL  PRIMARY KEY
  , system_message  TEXT  NOT NULL
  , user_message    TEXT  NOT NULL
  , content         TEXT  NOT NULL
  , content_html    TEXT  NOT NULL
  );
