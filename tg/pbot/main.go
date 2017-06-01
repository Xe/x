package main

import (
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Xe/uuid"
	"github.com/fogleman/primitive/primitive"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/telegram-bot-api.v4"

	// image formats
	_ "image/jpeg"
	_ "image/png"

	_ "github.com/hullerob/go.farbfeld"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	ps, ok := puushLogin(os.Getenv("PUUSH_KEY"))
	if !ok {
		log.Fatal("puush login failed")
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		err := renderImg(bot, ps, update)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "error: "+err.Error())
			log.Printf("error in processing message from %s: %v", update.Message.From.String(), err)
			bot.Send(msg)
		}
	}
}

func stepImg(img image.Image, count int) image.Image {
	bg := primitive.MakeColor(primitive.AverageImageColor(img))
	model := primitive.NewModel(img, bg, 512, runtime.NumCPU())

	for range make([]struct{}, count) {
		model.Step(primitive.ShapeTypeTriangle, 128, 0)
	}

	return model.Context.Image()
}

func renderImg(bot *tgbotapi.BotAPI, ps string, update tgbotapi.Update) error {
	msg := update.Message

	// ignore chats without photos
	if len(*msg.Photo) == 0 {
		return nil
	}

	p := *msg.Photo
	pho := p[len(p)-1]
	fu, err := bot.GetFileDirectURL(pho.FileID)
	if err != nil {
		return err
	}

	resp, err := http.Get(fu)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	img, ifmt, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	log.Printf("%s: image id %s loaded (%s)", msg.From, pho.FileID, ifmt)
	umsg := tgbotapi.NewMessage(update.Message.Chat.ID, "rendering... (may take a while)")
	bot.Send(umsg)

	before := time.Now()
	imgs := []image.Image{}

	for i := range make([]struct{}, 10) {
		log.Printf("%s: starting frame render", msg.From)
		imgs = append(imgs, stepImg(img, 150))
		log.Printf("%s: frame rendered", msg.From)

		umsg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("frame %d/10 rendered", i+1))
		bot.Send(umsg)
	}

	gpath := "./var/" + update.Message.From.String() + ".gif"
	err = primitive.SaveGIFImageMagick(gpath, imgs, 15, 15)
	if err != nil {
		return err
	}

	after := time.Now().Sub(before)

	buf, err := os.Open(gpath)
	if err != nil {
		return err
	}
	defer os.Remove(gpath)

	umsg = tgbotapi.NewMessage(update.Message.Chat.ID, "uploading (took "+after.String()+" to render)")
	bot.Send(umsg)

	furl, err := puush(ps, uuid.New()+".gif", buf)
	if err != nil {
		return err
	}

	omsg := tgbotapi.NewMessage(update.Message.Chat.ID, furl.String())
	_, err = bot.Send(omsg)
	if err != nil {
		return err
	}

	return nil
}
