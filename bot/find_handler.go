package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/admirallarimda/tgbotbase"
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type findHandler struct {
	tgbotbase.BaseHandler

	cardsDir string
	cache    *PicCache

	cardsByID   map[string]Card
	cardsByName map[string]Card
}

const dumpFilename = "all.dump.json"

var _ tgbotbase.IncomingMessageHandler = &findHandler{}

func NewFindHandler(cardsDir string, cache *PicCache) tgbotbase.IncomingMessageHandler {
	h := findHandler{
		cardsDir:    cardsDir,
		cache:       cache,
		cardsByID:   make(map[string]Card),
		cardsByName: make(map[string]Card),
	}
	return &h
}

func loadDump(dumpPath string) error {
	const url = "https://archive.scryfall.com/json/scryfall-all-cards.json"
	log.WithFields(log.Fields{"url": url, "dumpFile": dumpPath}).Info("loading new dump")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	dumpTmp := dumpPath + ".tmp"
	out, err := os.Create(dumpTmp)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	if err := os.Rename(dumpTmp, dumpPath); err != nil {
		return err
	}
	log.WithFields(log.Fields{"dumpFile": dumpPath}).Info("new dump has been downloaded")
	return nil
}

type Images struct {
	Normal string `json:"normal"`
}
type Card struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LocalName string `json:"printed_name"`
	Lang      string `json:"lang"`
	ImageURIs Images `json:"image_uris"`
}

var re = regexp.MustCompile("\\[\\[(.*)\\]\\]")

func (h *findHandler) Init(outMsgCh chan<- tgbotapi.Chattable, srvCh chan<- tgbotbase.ServiceMsg) tgbotbase.HandlerTrigger {
	dumpPath := path.Join(h.cardsDir, dumpFilename)
	os.MkdirAll(h.cardsDir, os.ModePerm)
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		log.WithFields(log.Fields{"dumpPath": dumpPath}).Info("dump is absent, loading")
		if err := loadDump(dumpPath); err != nil {
			panic(err)
		}

	}
	f, err := os.Open(dumpPath)
	if err != nil {
		panic(err)
	}
	log.WithFields(log.Fields{"dumpPath": dumpPath}).Info("decoding dump")
	dec := json.NewDecoder(f)
	_, err = dec.Token()
	if err != nil {
		panic(err)
	}
	for dec.More() {
		var c Card
		err := dec.Decode(&c)
		if err != nil {
			panic(err)
		}
		h.cardsByID[c.ID] = c
		names := []string{c.Name, c.LocalName}
		for _, n := range names {
			n := strings.ToLower(n)
			_, found := h.cardsByName[n]
			if found {
				if c.Lang == "en" {
					h.cardsByName[n] = c
				}
			} else {
				h.cardsByName[n] = c
			}
		}
	}
	_, err = dec.Token()
	if err != nil {
		panic(err)
	}
	log.WithFields(log.Fields{"cardsByID": len(h.cardsByID), "cardsByName": len(h.cardsByName)}).Info("decoding done")

	h.OutMsgCh = outMsgCh
	return tgbotbase.NewHandlerTrigger(re, []string{"find"})
}

func (h *findHandler) HandleOne(msg tgbotapi.Message) {
	cardname := ""
	if msg.IsCommand() {
		cardname = msg.CommandArguments()
	} else {
		cardname = re.FindStringSubmatch(msg.Text)[1]
	}
	log.WithFields(log.Fields{"cardname": cardname, "msg": msg.Text}).Info("message triggered")
	if c, found := h.cardsByName[strings.ToLower(strings.TrimSpace(cardname))]; found {
		picPath, err := h.cache.Get(c.ID, c.ImageURIs.Normal)
		if err != nil {
			log.WithFields(log.Fields{"id": c.ID, "err": err, "picPath": picPath}).Error("unable to get a picture from cache")
		}
		picMsg := tgbotapi.NewPhotoUpload(int64(msg.Chat.ID), picPath)
		picMsg.Caption = c.LocalName

		h.OutMsgCh <- picMsg
	} else {
		h.OutMsgCh <- tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("There's no card named %q", cardname))
	}
}

func (h *findHandler) Name() string {
	return "Find Handler"
}
