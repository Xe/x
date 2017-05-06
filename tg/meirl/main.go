package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/Xe/ln"
	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"github.com/turnage/graw"
	"github.com/turnage/graw/reddit"
	"gopkg.in/telegram-bot-api.v4"
)

const appid = "github.com/Xe/x/tg/meirl"
const version = "0.1-dev"

type config struct {
	RedditBotAdmin string   `env:"REDDIT_ADMIN_USERNAME,required"`
	Subreddits     []string `env:"SUBREDDITS,required"`

	TelegramToken     string `env:"TELEGRAM_TOKEN,required"`
	TelegramAdmin     string `env:"TELEGRAM_ADMIN,required"`
	TelegramChannelID int64  `env:"TELEGRAM_CHANNEL_ID,required"`
}

func main() {
	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "env.Parse"})
	}

	tg, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "tgbotapi.NewBotAPI"})
	}

	ln.Log(ln.F{"action": "telegram_active", "username": tg.Self.UserName})

	userAgent := fmt.Sprintf(
		"%s on %s %s:%s:%s (by /u/%s)",
		runtime.Version(), runtime.GOOS, runtime.GOARCH,
		appid, version, cfg.RedditBotAdmin,
	)

	rd, err := reddit.NewScript(userAgent, 5*time.Second)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "user_agent": userAgent})
	}
	_ = rd

	ln.Log(ln.F{"action": "reddit_connected", "user_agent": userAgent})

	a := &announcer{
		cfg: &cfg,
		tg:  tg,
	}

	stop, wait, err := graw.Scan(a, rd, graw.Config{Subreddits: cfg.Subreddits, Logger: log.New(os.Stderr, "", log.LstdFlags)})
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "graw.Scan", "subreddits": cfg.Subreddits})
	}
	defer stop()

	// This time, let's block so the bot will announce (ideally) forever.
	if err := wait(); err != nil {
		ln.Fatal(ln.F{"err": err, "action": "reddit_wait"})
	}
}

type announcer struct {
	cfg *config
	tg  *tgbotapi.BotAPI
}

func (a *announcer) Post(post *reddit.Post) error {
	msg := tgbotapi.NewMessage(a.cfg.TelegramChannelID, fmt.Sprintf("me irl\n%s\n(https://reddit.com%s by /u/%s)", post.URL, post.Permalink, post.Author))
	_, err := a.tg.Send(msg)
	if err != nil {
		ln.Error(err, ln.F{"err": err, "action": "telegram_post"})
		return err
	}

	ln.Log(ln.F{"action": "new_post", "url": post.URL, "permalink": post.Permalink, "redditor": post.Author})

	return nil
}
