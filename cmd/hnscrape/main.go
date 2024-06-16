package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"within.website/x/internal"
)

var (
	cacheFolder = flag.String("cache-folder", "./var/hn", "Folder to cache items in")
	hnUser      = flag.String("hn-user", "xena", "Hacker News user to scrape")
	scrapeDelay = flag.Duration("scrape-delay", 50*time.Millisecond, "Delay between scraping items")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := ControlCContext()
	defer cancel()

	slog.Info("starting hnscrape", "scrapeDelay", scrapeDelay.String(), "hnUser", *hnUser)

	hn := NewHNClient(*scrapeDelay)

	if *cacheFolder != "" {
		slog.Info("caching items to", "cacheFolder", *cacheFolder)
		os.MkdirAll(*cacheFolder, 0755)
		os.MkdirAll(filepath.Join(*cacheFolder, "items"), 0755)
		os.MkdirAll(filepath.Join(*cacheFolder, "indices"), 0755)
		hn = hn.WithCacheFolder(*cacheFolder)
	}

	u, err := hn.GetUser(ctx, *hnUser)
	if err != nil {
		slog.Error("failed to get user", "err", err, "user", *hnUser)
		os.Exit(1)
	}

	slog.Info("got user", "user", u)

	// itemsCommentedIn := make([]int, 0)
	//
	// wg := sync.WaitGroup{}
	//
	// wg.Add(len(u.Submitted))
	//
	// for _, itemID := range u.Submitted {
	// 	itemID := itemID
	//
	// 	item, err := hn.GetItem(ctx, itemID)
	// 	if err != nil {
	// 		slog.Error("failed to get item", "err", err, "itemID", itemID)
	// 		continue
	// 	}
	//
	// 	slog.Debug("got item", "item", item.ID)
	//
	// 	go func(hn *HNClient, item *HNItem) {
	// 		defer wg.Done()
	//
	// 		if item.Type != "comment" {
	// 			return
	// 		}
	//
	// 		if item.Parent == nil {
	// 			return
	// 		}
	//
	// 		/*
	// 			if len(item.Kids) != 0 {
	// 				slog.Info("getting comment kids", "item", item.ID)
	// 				for _, kid := range item.Kids {
	// 					kid, err := hn.GetItem(ctx, kid)
	// 					if err != nil {
	// 						slog.Error("failed to get kid", "err", err, "kid", kid)
	// 						continue
	// 					}
	//
	// 					slog.Info("got kid", "kid", kid.ID)
	// 				}
	// 			}*/
	//
	// 		slog.Debug("getting comment parent", "parent", item.Parent)
	//
	// 		parent, err := hn.GetItem(ctx, *item.Parent)
	// 		if err != nil {
	// 			if err == context.Canceled {
	// 				return
	// 			}
	// 			slog.Error("failed to get parent", "err", err, "parent", item.Parent)
	// 			return
	// 		}
	//
	// 		slog.Debug("got parent", "parent", parent.ID)
	// 		/*
	// 			for _, kid := range parent.Kids {
	// 				kid, err := hn.GetItem(ctx, kid)
	// 				if err != nil {
	// 					slog.Error("failed to get kid", "err", err, "kid", kid)
	// 					continue
	// 				}
	//
	// 				slog.Info("got kid", "kid", kid.ID)
	// 			}*/
	// 	}(hn, item)
	//
	// 	/*
	// 		if item.Type == "comment" {
	// 			ultimateParent, err := hn.GetUltimateParent(ctx, item.ID)
	// 			if err != nil {
	// 				if err == context.Canceled {
	// 					break
	// 				}
	// 				slog.Error("failed to get ultimate parent", "err", err, "item", item.ID)
	// 				continue
	// 			}
	// 			itemsCommentedIn = append(itemsCommentedIn, ultimateParent.ID)
	// 		}
	// 	*/
	// }
	//
	// wg.Wait()

	/*
		slog.Info("done", "itemsCommentedIn", len(itemsCommentedIn))

		fout, err := os.Create(filepath.Join(*cacheFolder, "indices", "itemsCommentedIn"))
		if err != nil {
			slog.Error("failed to create itemsCommentedIn file", "err", err)
			os.Exit(1)
		}
		defer fout.Close()

		if err := json.NewEncoder(fout).Encode(itemsCommentedIn); err != nil {
			slog.Error("failed to write itemsCommentedIn", "err", err)
			os.Exit(1)
		}
	*/
}

func ControlCContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
		cancel()
		<-sc
		os.Exit(1)
	}()

	return ctx, cancel
}
