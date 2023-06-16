CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    discord_server TEXT NOT NULL,
    discord_id TEXT NOT NULL,
    revolt_server TEXT NOT NULL,
    revolt_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    hoist BOOLEAN NOT NULL
);