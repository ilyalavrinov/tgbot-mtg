package bot

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/admirallarimda/tgbotbase"
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type findHandler struct {
	tgbotbase.BaseHandler

	cardsDir string
	cache    *PicCache
}

const dumpFilename = "all.dump.json"

var _ tgbotbase.IncomingMessageHandler = &findHandler{}

func NewFindHandler(cardsDir string, cache *PicCache) tgbotbase.IncomingMessageHandler {
	h := findHandler{
		cardsDir: cardsDir,
		cache:    cache,
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

type Cards []Card

type Card struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	PermalinkURI    string `json:"permalink_uri"`
	UpdatedAt       string `json:"updated_at"`
	CompressedSize  int    `json:"compressed_size"`
	ContentType     string `json:"content_type"`
	ContentEncoding string `json:"content_encoding"`
}

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
	var cards Cards
	log.WithFields(log.Fields{"dumpPath": dumpPath, "size": len(data)}).Info("unmarshalling dump")
	json.Unmarshal(data, &cards)
	log.WithFields(log.Fields{"cards": len(cards)}).Info("unmarshalling done")

	h.OutMsgCh = outMsgCh
	return tgbotbase.NewHandlerTrigger(nil, []string{"find"})
}

func (h *findHandler) HandleOne(msg tgbotapi.Message) {

}

func (h *findHandler) Name() string {
	return "Find Handler"
}
