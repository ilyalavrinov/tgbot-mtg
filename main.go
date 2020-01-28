package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	"./bot"
	"github.com/admirallarimda/tgbotbase"
	"gopkg.in/gcfg.v1"
)

var argCfg = flag.String("cfg", "./mtgbot.cfg", "path to config")

type config struct {
	tgbotbase.Config

	Cards struct {
		ScryfallDumpDir string
	}

	Cache struct {
		Dir string
	}
}

func main() {
	flag.Parse()

	var cfg config

	if err := gcfg.ReadFileInto(&cfg, *argCfg); err != nil {
		log.WithFields(log.Fields{"filepath": *argCfg, "error": err}).Fatal("Config parse failed")
	}

	tgbot := tgbotbase.NewBot(tgbotbase.Config{TGBot: cfg.TGBot, Proxy_SOCKS5: cfg.Proxy_SOCKS5})
	if cfg.Cache.Dir == "" {
		cfg.Cache.Dir = "./piccache"
	}
	if cfg.Cards.ScryfallDumpDir == "" {
		cfg.Cards.ScryfallDumpDir = "./scryfall"
	}

	tgbot.AddHandler(tgbotbase.NewIncomingMessageDealer(bot.NewFindHandler(cfg.Cards.ScryfallDumpDir, bot.NewPicCache(cfg.Cache.Dir))))

	log.Info("Starting bot")
	tgbot.Start()
	log.Info("Stopping bot")
}
