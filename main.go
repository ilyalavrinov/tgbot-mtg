package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	"github.com/admirallarimda/tgbotbase"
	"github.com/ilyalavrinov/tgbot-mtg/bot"
	"gopkg.in/gcfg.v1"
)

var argCfg = flag.String("cfg", "./mtgbot.cfg", "path to config")

type config struct {
	tgbotbase.Config

	Cache struct {
		Dir string
	}
}

func main() {
	flag.Parse()

	var cfg config

	if err := gcfg.ReadFileInto(&cfg, *argCfg); err != nil {
		log.WithFields(log.Fields{"filepath": argCfg, "error": err}).Fatal("Config parse failed")
	}

	tgbot := tgbotbase.NewBot(tgbotbase.Config{TGBot: cfg.TGBot})

	tgbot.AddHandler(tgbotbase.NewIncomingMessageDealer(bot.NewFindHandler(bot.NewPicCache(cfg.Cache.Dir))))

	log.Info("Starting bot")
	tgbot.Start()
	log.Info("Stopping bot")
}
