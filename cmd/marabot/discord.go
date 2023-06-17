package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"within.website/ln"
	"within.website/ln/opname"
)

func importDiscordData(ctx context.Context, db *sql.DB, dg *discordgo.Session) error {
	ctx = opname.With(ctx, "import-discord-data")
	channels, err := dg.GuildChannels(*furryholeDiscord)
	if err != nil {
		return err
	}

	for _, ch := range channels {
		if _, err := db.Exec("INSERT INTO discord_channels (id, guild_id, name, topic, nsfw) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, topic = EXCLUDED.topic, nsfw = EXCLUDED.nsfw", ch.ID, ch.GuildID, ch.Name, ch.Topic, ch.NSFW); err != nil {
			ln.Error(ctx, err, ln.F{"channel_name": ch.Name, "channel_id": ch.ID})
			continue
		}
	}

	roles, err := dg.GuildRoles(*furryholeDiscord)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if _, err := db.ExecContext(ctx, "INSERT INTO discord_roles (guild_id, id, name, color, hoist, position) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, color = EXCLUDED.color, position = EXCLUDED.position", furryholeDiscord, role.ID, role.Name, fmt.Sprintf("#%06x", role.Color), role.Hoist, role.Position); err != nil {
			ln.Error(ctx, err, ln.Action("inserting role"))
			continue
		}
	}

	// https://cdn.discordapp.com/emojis/664686615616290816.webp?size=240&quality=lossless
	emoji, err := dg.GuildEmojis(*furryholeDiscord)
	if err != nil {
		return err
	}
	for _, emoji := range emoji {
		if _, err := db.ExecContext(ctx, "INSERT INTO discord_emoji (id, guild_id, name, url) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url", emoji.ID, furryholeDiscord, emoji.Name, fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.webp?size=240&quality=lossless", emoji.ID)); err != nil {
			ln.Error(ctx, err, ln.Action("inserting emoji"))
			continue
		}
	}

	return nil
}
