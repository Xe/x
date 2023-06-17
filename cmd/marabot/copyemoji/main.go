package main

import (
	"context"
	"database/sql"
	"flag"
	"io"
	"net/http"

	_ "modernc.org/sqlite"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web/revolt"
)

var (
	dbFile                = flag.String("db-file", "../marabot.db", "Path to the database file")
	furryholeDiscord      = flag.String("furryhole-discord", "192289762302754817", "Discord channel ID for furryhole")
	furryholeRevolt       = flag.String("furryhole-revolt", "01FEXZ1XPWMEJXMF836FP16HB8", "Revolt channel ID for furryhole")
	revoltEmail           = flag.String("revolt-email", "", "Email for Revolt")
	revoltPassword        = flag.String("revolt-password", "", "Password for Revolt")
	revoltAPIServer       = flag.String("revolt-api-server", "https://api.revolt.chat", "API server for Revolt")
	revoltWebsocketServer = flag.String("revolt-ws-server", "wss://ws.revolt.chat", "Websocket server for Revolt")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(opname.With(context.Background(), "marabot.copyemoji"))
	defer cancel()

	ln.Log(ctx, ln.Action("starting up"))

	db, err := sql.Open("sqlite", *dbFile)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("opening sqlite database"))
	}
	defer db.Close()

	cli, err := revolt.NewWithEndpoint("", *revoltAPIServer, *revoltWebsocketServer)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("creating revolt client"))
	}
	cli.SelfBot = &revolt.SelfBot{
		Email:    *revoltEmail,
		Password: *revoltPassword,
	}

	if err := cli.Auth(ctx, "marabot-copyemoji"); err != nil {
		ln.FatalErr(ctx, err, ln.Action("authing to revolt"))
	}

	rows, err := db.QueryContext(ctx, "SELECT id, name, url FROM discord_emoji WHERE guild_id = ? AND id NOT IN ( SELECT discord_id FROM revolt_discord_emoji )", furryholeDiscord)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("querying discord_emoji"))
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, url string
		if err := rows.Scan(&id, &name, &url); err != nil {
			ln.FatalErr(ctx, err, ln.Action("scanning discord_emoji"))
		}

		resp, err := http.Get(url)
		if err != nil {
			ln.Error(ctx, err, ln.F{"url": url}, ln.Action("downloading emoji"))
			continue
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			ln.Error(ctx, err, ln.F{"url": url}, ln.Action("reading emoji body"))
			continue
		}

		uploadID, err := cli.Upload(ctx, "emojis", name+".webp", data)
		if err != nil {
			ln.Error(ctx, err, ln.F{"url": url}, ln.Action("uploading emoji"))
			continue
		}

		emoji, err := cli.CreateEmoji(ctx, uploadID, revolt.CreateEmoji{
			Name: name,
			NSFW: false,
			Parent: revolt.Parent{
				ID:   *furryholeRevolt,
				Type: "Server",
			},
		})
		if err != nil {
			ln.Error(ctx, err, ln.F{"url": url}, ln.Action("creating emoji on revolt"))
			continue
		}

		if _, err := db.ExecContext(ctx, "INSERT INTO revolt_emoji (id, server_id, name, url) VALUES (?, ?, ?, ?)", emoji.ID, furryholeRevolt, emoji.Name, emoji.URL); err != nil {
			ln.Error(ctx, err, ln.F{"id": emoji.ID}, ln.Action("inserting emoji"))
			continue
		}

		if _, err := db.ExecContext(ctx, "INSERT INTO revolt_discord_emoji(revolt_id, discord_id, name) VALUES (?, ?, ?)", emoji.ID, id, name); err != nil {
			ln.Error(ctx, err, ln.F{"id": emoji.ID}, ln.Action("inserting emoji join record"))
			continue
		}

		ln.Log(ctx, ln.Info("created emoji"), ln.F{"id": emoji.ID, "name": emoji.Name})
	}
}
