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
	ID         string `json:"id"`
	Name       string `json:"name"`
	LocalName  string `json:"printed_name"`
	Lang       string `json:"lang"`
	ImageURIs  Images `json:"image_uris"`
	URI        string `json:"uri"`
	RulingsURI string `json:"rulings_uri"`
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
		if c.LocalName == "" {
			c.LocalName = c.Name
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
	req := ""
	if msg.IsCommand() {
		req = msg.CommandArguments()
	} else {
		req = re.FindStringSubmatch(msg.Text)[1]
	}
	log.WithFields(log.Fields{"req": req, "msg": msg.Text}).Info("message triggered")
	req = strings.Trim(req, " \n\t[]")
	cardname := strings.ToLower(strings.Trim(req, "$#"))
	if req == "" || cardname == "" {
		return
	}
	c, found := h.cardsByName[cardname]
	if !found {
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("There's no card named %q", cardname))
		reply.ReplyToMessageID = msg.MessageID
		h.OutMsgCh <- reply
		return
	}
	switch string(req[0]) {
	case "$":
		h.handlePrice(c, msg)
	case "#":
		h.handleRulings(c, msg)
	default:
		h.handleCard(c, msg)
	}
}

func (h *findHandler) handleCard(c Card, msg tgbotapi.Message) {
	picPath, err := h.cache.Get(c.ID, c.ImageURIs.Normal)
	if err != nil {
		log.WithFields(log.Fields{"id": c.ID, "err": err, "picPath": picPath}).Error("unable to get a picture from cache")
	}
	picMsg := tgbotapi.NewPhotoUpload(int64(msg.Chat.ID), picPath)
	picMsg.Caption = c.LocalName
	picMsg.ReplyToMessageID = msg.MessageID

	h.OutMsgCh <- picMsg
}

type cardFull struct {
	Prices struct {
		USD     string
		USDFoil string `json:"usd_foil"`
		EUR     string
	}
}

func (h *findHandler) handlePrice(c Card, msg tgbotapi.Message) {
	resp, err := http.Get(c.URI)
	if err != nil {
		log.WithFields(log.Fields{"cardID": c.ID, "URI": c.URI, "err": err}).Error("cannot load info from card URI")
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"cardID": c.ID, "URI": c.URI, "err": err}).Error("cannot read API response")
		return
	}

	var cFull cardFull
	if err = json.Unmarshal(body, &cFull); err != nil {
		log.WithFields(log.Fields{"cardID": c.ID, "URI": c.URI, "err": err}).Error("cannot unmarshal full card info")
		return
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Prices for %q:\nUSD: %s\nUSD Foil: %s\nEUR: %s", c.LocalName, cFull.Prices.USD, cFull.Prices.USDFoil, cFull.Prices.EUR))
	reply.ReplyToMessageID = msg.MessageID
	h.OutMsgCh <- reply
}

type ruling struct {
	PublishedAt string `json:"published_at"`
	Comment     string
}
type rulings struct {
	Data []ruling
}

func (h *findHandler) handleRulings(c Card, msg tgbotapi.Message) {
	resp, err := http.Get(c.RulingsURI)
	if err != nil {
		log.WithFields(log.Fields{"cardID": c.ID, "URI": c.URI, "err": err}).Error("cannot load rulings")
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"cardID": c.ID, "URI": c.URI, "err": err}).Error("cannot read rulings")
		return
	}

	var rules rulings
	if err = json.Unmarshal(body, &rules); err != nil {
		log.WithFields(log.Fields{"cardID": c.ID, "URI": c.URI, "err": err}).Error("cannot unmarshal rulings")
		return
	}

	replyTxt := ""
	if len(rules.Data) == 0 {
		replyTxt = fmt.Sprintf("Card %q does not have specific rulings", c.LocalName)
	} else {
		for _, d := range rules.Data {
			replyTxt = fmt.Sprintf("%s%s: %s\n", replyTxt, d.PublishedAt, d.Comment)
		}
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, replyTxt)
	reply.ReplyToMessageID = msg.MessageID
	h.OutMsgCh <- reply
}

func (h *findHandler) Name() string {
	return "Find Handler"
}
