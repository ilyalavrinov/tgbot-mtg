package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/admirallarimda/tgbotbase"
	"github.com/ilyalavrinov/mtgbulkbuy/pkg/mtgbulk"
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type edhrecCmdrDailyUpdate struct {
	cardname    string
	url, picUrl string
	rankInfo    string
	salt        float32
	minPrice    price
}

type edhrecCmdrDailyHandler struct {
	tgbotbase.BaseHandler
	props tgbotbase.PropertyStorage
	cron  tgbotbase.Cron

	updates chan edhrecCmdrDailyUpdate
}

var _ tgbotbase.BackgroundMessageHandler = &edhrecCmdrDailyHandler{}

func NewEdhrecCmdrDailyHandler(cron tgbotbase.Cron,
	props tgbotbase.PropertyStorage) tgbotbase.BackgroundMessageHandler {
	h := &edhrecCmdrDailyHandler{
		props: props,
		cron:  cron,
	}
	h.updates = make(chan edhrecCmdrDailyUpdate, 0)
	return h
}

func (h *edhrecCmdrDailyHandler) Init(outMsgCh chan<- tgbotapi.Chattable, srvCh chan<- tgbotbase.ServiceMsg) {
	h.OutMsgCh = outMsgCh
}

func (h *edhrecCmdrDailyHandler) Run() {
	chatsToNotify := make([]tgbotbase.ChatID, 0)
	props, _ := h.props.GetEveryHavingProperty("edhrecCmdrDailyNotify")
	for _, prop := range props {
		if (prop.User != 0) && (tgbotbase.ChatID(prop.User) != prop.Chat) {
			continue
		}
		chatsToNotify = append(chatsToNotify, prop.Chat)
	}

	prevDealName, err := h.props.GetProperty("edhrecCmdrDailyLast", 0, 0)
	if err != nil {
		panic(fmt.Sprintf("Could not get last mtgsale deal, err: %s", err))
	}

	go func() {
		data := edhrecCmdrDailyUpdate{}
		for {
			select {
			case data = <-h.updates:
				if data.cardname == prevDealName {
					continue
				}

				prevDealName = data.cardname
				h.props.SetPropertyForUserInChat("edhrecCmdrDailyLast", 0, 0, prevDealName)

				picFName, err := loadPicToTmp(data.picUrl, "mtgbot-edhrec-")
				if err != nil {
					log.Errorf("Could not load edhrec daily cmdr pic, err: %s", err)
					continue
				}

				saltScore := fmt.Sprintf("Salt score: %.2f", data.salt)
				saltScore = escapeMarkdown(saltScore)
				data.rankInfo = escapeMarkdown(data.rankInfo)
				text := fmt.Sprintf("Commander of the day\n[%s](%s)\n%s\n%s", data.cardname, data.url, data.rankInfo, saltScore)
				if data.minPrice.Price != 0 {
					text = fmt.Sprintf("%s\n%s", text, formatPrice("min", data.minPrice))
				}
				for _, chatID := range chatsToNotify {
					msg := tgbotapi.NewPhotoUpload(int64(chatID), picFName)
					msg.Caption = text
					msg.ParseMode = "MarkdownV2"
					h.OutMsgCh <- msg
				}
			}
		}
	}()

	h.cron.AddJob(time.Now(), &edhrecCmdrDailyJob{updates: h.updates})
}

func (h *edhrecCmdrDailyHandler) Name() string {
	return "mtgsale new deal"
}

type edhrecCmdrDailyJob struct {
	updates chan<- edhrecCmdrDailyUpdate
}

func (job *edhrecCmdrDailyJob) Do(scheduledWhen time.Time, cron tgbotbase.Cron) {
	defer cron.AddJob(scheduledWhen.Add(30*time.Minute), job)

	resp, err := http.Get("https://edhrec.com/api/daily/")
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Unable to get edhrec daily api")
		return
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var dailyData struct {
		Daily struct {
			Name  string
			URL   string
			Image string
		}
	}
	err = decoder.Decode(&dailyData)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Unable to decode edhrec daily api")
		return
	}

	curCmdr := edhrecCmdrDailyUpdate{}
	curCmdr.cardname = dailyData.Daily.Name
	curCmdr.url = "https://edhrec.com" + dailyData.Daily.URL
	curCmdr.picUrl = dailyData.Daily.Image

	{
		rankDataURL := "https://edhrec-json.s3.amazonaws.com/en" + dailyData.Daily.URL + ".json"
		resp, err := http.Get(rankDataURL)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Unable to get cmdr data")
			return
		}
		defer resp.Body.Close()

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Unable to read cmdr data")
			return
		}

		var cmdrData struct {
			Container struct {
				Json_dict struct {
					Card struct {
						Label string
						Salt  float32
					}
				} `json:"json_dict"`
			}
		}
		err = json.Unmarshal(b, &cmdrData)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Unable to decode cmdr data")
			return
		}
		curCmdr.rankInfo = cmdrData.Container.Json_dict.Card.Label
		curCmdr.salt = cmdrData.Container.Json_dict.Card.Salt
	}

	log.WithFields(log.Fields{
		"card":      curCmdr.cardname,
		"url":       curCmdr.url,
		"picUrl":    curCmdr.picUrl,
		"rankInfo":  curCmdr.rankInfo,
		"saltScore": curCmdr.salt}).Debug("scrapped edhrec cmdr")

	{
		req := mtgbulk.NewNamesRequest()
		req.Cards[curCmdr.cardname] = 1
		resp, err := mtgbulk.ProcessByNames(req)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Unable to get prices")
			return
		}
		minPriceRes := resp.MinPricesRule[curCmdr.cardname][0]
		curCmdr.minPrice.Price = int(minPriceRes.Price)
		curCmdr.minPrice.Seller = minPriceRes.Trader
		curCmdr.minPrice.URL = minPriceRes.URL
	}

	job.updates <- curCmdr
}
