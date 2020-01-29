package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"

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

func (h *findHandler) loadCards(cards []Card) {
	for _, c := range cards {
		h.cardsByID[c.ID] = c
		h.cardsByName[c.Name] = c
		h.cardsByName[c.LocalName] = c
	}
}

type Card struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	LocalName   string `json:"printed_name"`
	Lang        string `json:"lang"`
	ScryfallURL string `json:"scryfall_uri"`
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
	data, _ := ioutil.ReadAll(f)
	var cards []Card
	log.WithFields(log.Fields{"dumpPath": dumpPath, "size": len(data)}).Info("unmarshalling dump")
	json.Unmarshal(data, &cards)
	log.WithFields(log.Fields{"cards": len(cards)}).Info("unmarshalling done")

	h.loadCards(cards)

	h.OutMsgCh = outMsgCh
	return tgbotbase.NewHandlerTrigger(re, []string{"find"})
}

func (h *findHandler) HandleOne(msg tgbotapi.Message) {
	cardname := ""
	if msg.IsCommand() {
		cardname = msg.CommandArguments()
	} else {
		cardname = re.FindString(msg.Text)
	}
	log.WithFields(log.Fields{"cardname": cardname, "msg": msg.Text}).Info("message triggered")
	if c, found := h.cardsByName[cardname]; found {
		h.OutMsgCh <- tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("card for name %q has been found. URL: %s", cardname, c.ScryfallURL))
	} else {
		h.OutMsgCh <- tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("could not find a card for name %q", cardname))
	}
}

func (h *findHandler) Name() string {
	return "Find Handler"
}
