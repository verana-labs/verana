package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/verana-labs/verana/tools/cipher/internal/bot"
	"github.com/verana-labs/verana/tools/cipher/internal/config"
	"github.com/verana-labs/verana/tools/cipher/internal/executor"
	ghclient "github.com/verana-labs/verana/tools/cipher/internal/github"
	"github.com/verana-labs/verana/tools/cipher/internal/state"
	"github.com/verana-labs/verana/tools/cipher/internal/watcher"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[cipher] config: %v", err)
	}

	st := state.New(cfg.StateFile)
	gh := ghclient.New(cfg)
	exec := executor.New(cfg, gh, st)

	b, err := bot.New(cfg, exec, st)
	if err != nil {
		log.Fatalf("[cipher] bot: %v", err)
	}

	w := watcher.New(cfg, gh, st, exec, b.Notify)
	bot.WatcherRef = w
	go w.Start()

	if err := b.Open(); err != nil {
		log.Fatalf("[cipher] open: %v", err)
	}
	defer b.Close()

	log.Println("[cipher] running. Ctrl+C to stop.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
	log.Println("[cipher] shutting down.")
}
