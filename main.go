package main

import (
	"flag"

	"github.com/admirallarimda/tgbotbase"
	"github.com/ilyalavrinov/tgbot-mtg/bot"
	"gopkg.in/gcfg.v1"

	log "github.com/sirupsen/logrus"
)

var argCfg = flag.String("cfg", "./mtgbot.cfg", "path to config")

type config struct {
	tgbotbase.Config
	Redis tgbotbase.RedisConfig

	Cards struct {
		ScryfallDumpDir string
	}

	Cache struct {
		Dir string
	}
}

func main() {
	//	log.SetLevel(log.DebugLevel)
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

	cron := tgbotbase.NewCron()
	pool := tgbotbase.NewRedisPool(cfg.Redis)
	props := tgbotbase.NewRedisPropertyStorage(pool)

	tgbot.AddHandler(tgbotbase.NewIncomingMessageDealer(bot.NewFindHandler(cfg.Cards.ScryfallDumpDir, bot.NewPicCache(cfg.Cache.Dir))))
	tgbot.AddHandler(tgbotbase.NewBackgroundMessageDealer(bot.NewMtgsaleDealHandler(cron, props)))

	log.Info("Starting bot")
	tgbot.Start()
	log.Info("Stopping bot")
}
