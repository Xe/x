PRAGMA journal_mode=WAL;

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

CREATE TABLE IF NOT EXISTS discord_channels (
    id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    name TEXT NOT NULL,
    topic TEXT NOT NULL,
    nsfw BOOLEAN NOT NULL
);

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

CREATE TABLE IF NOT EXISTS irc_messages (
    id SERIAL PRIMARY KEY,
    nick TEXT NOT NULL,
    user TEXT NOT NULL,
    host TEXT NOT NULL,
    channel TEXT NOT NULL,
    content TEXT NOT NULL,
    tags TEXT NOT NULL
);
