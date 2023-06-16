package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web/revolt"
)

var (
	dbFile                = flag.String("db-file", "marabot.db", "Path to the database file")
	discordToken          = flag.String("discord-token", "", "Discord bot token")
	revoltToken           = flag.String("revolt-token", "", "Revolt bot token")
	revoltAPIServer       = flag.String("revolt-api-server", "https://api.revolt.chat", "API server for Revolt")
	revoltWebsocketServer = flag.String("revolt-ws-server", "wss://ws.revolt.chat", "Websocket server for Revolt")

	//go:embed schema.sql
	dbSchema string
)

const (
	furryholeDiscord = "192289762302754817"
	furryholeRevolt  = "01H2VRKJFPYPEAE438B6JRFSCP"
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(opname.With(context.Background(), "marabot"))
	defer cancel()

	ln.Log(ctx, ln.Action("starting up"))

	// Init a new client.
	client := revolt.NewWithEndpoint(*revoltToken, *revoltAPIServer, *revoltWebsocketServer)

	mr := &MaraRevolt{
		cli: client,
	}

	client.Connect(ctx, mr)

	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("creating discord client"))
	}

	if err := dg.Open(); err != nil {
		ln.FatalErr(ctx, err, ln.Action("opening discord client"))
	}
	defer dg.Close()

	db, err := sql.Open("sqlite", *dbFile)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("opening sqlite database"))
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, dbSchema); err != nil {
		ln.FatalErr(ctx, err, ln.Action("running database schema"))
	}

	// roles, err := dg.GuildRoles(furryholeDiscord)
	// if err != nil {
	// 	ln.FatalErr(ctx, err, ln.Action("getting guild roles"))
	// }

	// for _, role := range roles {
	// 	if role.Name == "@everyone" {
	// 		continue
	// 	}
	// 	ln.Log(ctx, ln.Info("role"), ln.F{"name": role.Name, "id": role.ID, "color": fmt.Sprintf("#%06x", role.Color)})

	// 	id, err := client.ServerCreateRole(ctx, furryholeRevolt, role.Name)
	// 	if err != nil {
	// 		ln.Error(ctx, err, ln.Action("creating role"))
	// 		continue
	// 	}

	// 	if err := client.ServerEditRole(ctx, furryholeRevolt, id, &revolt.EditRole{
	// 		Color: fmt.Sprintf("#%06x", role.Color),
	// 		Hoist: role.Hoist,
	// 		Rank:  250 - role.Position,
	// 	}); err != nil {
	// 		ln.Error(ctx, err, ln.Action("editing role"))
	// 		continue
	// 	}

	// 	if _, err := db.ExecContext(ctx, "INSERT INTO roles (discord_server, discord_id, revolt_server, revolt_id, name, color, hoist) VALUES (?, ?, ?, ?, ?, ?, ?)", furryholeDiscord, role.ID, furryholeRevolt, id, role.Name, fmt.Sprintf("#%06x", role.Color), role.Hoist); err != nil {
	// 		ln.Error(ctx, err, ln.Action("inserting role"))
	// 		continue
	// 	}
	// }

	// Wait for close.
	sc := make(chan os.Signal, 1)

	signal.Notify(
		sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-sc:
			ln.Log(ctx, ln.Info("shutting down"))
			cancel()
			time.Sleep(150 * time.Millisecond)
			return
		}
	}
}

type MaraRevolt struct {
	cli *revolt.Client
	revolt.NullHandler
}

func (m *MaraRevolt) MessageCreate(ctx context.Context, msg *revolt.Message) error {
	if msg.Content == "!ping" {
		sendMsg := &revolt.SendMessage{
			Masquerade: &revolt.Masquerade{
				Name:      "cipra",
				AvatarURL: "https://cdn.xeiaso.net/avatar/" + internal.Hash("cipra", "yasomi"),
			},
		}
		sendMsg.SetContent("🏓 Pong!")

		if _, err := m.cli.MessageReply(ctx, msg.ChannelId, msg.ID, true, sendMsg); err != nil {
			return err
		}
	}
	return nil
}
