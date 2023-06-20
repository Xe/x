PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA wal_autocheckpoint = 0;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS discord_roles (
    id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    hoist BOOLEAN NOT NULL,
    position INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS discord_users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    avatar_url TEXT NOT NULL,
    accent_color INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS discord_users_avatar_url
  ON discord_users(avatar_url);

CREATE TABLE IF NOT EXISTS discord_messages (
    id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    content TEXT,
    created_at TEXT NOT NULL,
    edited_at TEXT,
    webhook_id TEXT
);

CREATE INDEX IF NOT EXISTS discord_messages_author_id
  ON discord_messages(author_id);
  
CREATE INDEX IF NOT EXISTS discord_messages_guild_id
  ON discord_messages(guild_id);
  
CREATE INDEX IF NOT EXISTS discord_messages_channel_id
  ON discord_messages(channel_id);

CREATE TABLE IF NOT EXISTS discord_webhook_message_info (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    avatar_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS discord_attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    url TEXT NOT NULL,
    proxy_url TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    "size" INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS discord_attachments_message_id
  ON discord_attachments(message_id);

CREATE INDEX IF NOT EXISTS discord_attachments_url
  ON discord_attachments(url);

CREATE TABLE IF NOT EXISTS discord_channels (
    id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    name TEXT NOT NULL,
    topic TEXT NOT NULL,
    nsfw BOOLEAN NOT NULL
);

CREATE INDEX IF NOT EXISTS discord_channels_guild_id
  ON discord_channels(guild_id);

CREATE TABLE IF NOT EXISTS discord_guilds (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    icon_url TEXT NOT NULL,
    banner_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS discord_emoji (
    id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS discord_emoji_url
  ON discord_emoji(url);

CREATE TABLE IF NOT EXISTS irc_messages (
    id SERIAL PRIMARY KEY,
    nick TEXT NOT NULL,
    user TEXT NOT NULL,
    host TEXT NOT NULL,
    channel TEXT NOT NULL,
    content TEXT NOT NULL,
    tags TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS irc_messages_channel
  ON irc_messages(channel);

CREATE TABLE IF NOT EXISTS revolt_channels (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS revolt_emoji (
    id TEXT PRIMARY KEY,
    server_id TEXT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS revolt_emoji_url
  ON revolt_emoji(url);

CREATE TABLE IF NOT EXISTS revolt_discord_emoji (
    revolt_id TEXT NOT NULL,
    discord_id TEXT NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS revolt_messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS revolt_message_masquerade (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    avatar_url TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS revolt_message_masquerade_avatar_url
  ON revolt_message_masquerade(avatar_url);

CREATE TABLE IF NOT EXISTS revolt_servers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS revolt_users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    avatar_url TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS revolt_users_avatar_url
  ON revolt_users(avatar_url);

CREATE TABLE IF NOT EXISTS revolt_attachments (
    id TEXT PRIMARY KEY,
    tag TEXT NOT NULL,
    message_id TEXT,
    url TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    "size" INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS revolt_attachments_message_id
  ON revolt_attachments(message_id);

CREATE INDEX IF NOT EXISTS revolt_attachments_url
  ON revolt_attachments(url);

CREATE TABLE IF NOT EXISTS s3_uploads (
    id TEXT PRIMARY KEY, -- sha512 of file contents
    url TEXT NOT NULL,
    kind TEXT NOT NULL,
    content_type TEXT NOT NULL,
    created_at TEXT NOT NULL,
    message_id TEXT
);

CREATE INDEX IF NOT EXISTS s3_uploads_url
  ON s3_uploads(url);

CREATE INDEX IF NOT EXISTS s3_uploads_message_id
  ON s3_uploads(message_id);
