package bot

import (
	"fmt"
	"time"

	"github.com/admirallarimda/tgbotbase"
	"github.com/gocolly/colly"
	log "github.com/sirupsen/logrus"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type mtgsaleDealUpdate struct {
	cardname           string
	url, picUrl        string
	priceNew, priceOld string
}

type mtgSaleDealHandler struct {
	tgbotbase.BaseHandler
	props tgbotbase.PropertyStorage
	cron  tgbotbase.Cron

	updates chan mtgsaleDealUpdate
}

var _ tgbotbase.BackgroundMessageHandler = &mtgSaleDealHandler{}

func NewMtgsaleDealHandler(cron tgbotbase.Cron,
	props tgbotbase.PropertyStorage) tgbotbase.BackgroundMessageHandler {
	h := &mtgSaleDealHandler{
		props: props,
		cron:  cron,
	}
	h.updates = make(chan mtgsaleDealUpdate, 0)
	return h
}

func (h *mtgSaleDealHandler) Init(outMsgCh chan<- tgbotapi.Chattable, srvCh chan<- tgbotbase.ServiceMsg) {
	h.OutMsgCh = outMsgCh
}

func (h *mtgSaleDealHandler) Run() {
	chatsToNotify := make([]tgbotbase.ChatID, 0)
	props, _ := h.props.GetEveryHavingProperty("mtgsaleDealNotify")
	for _, prop := range props {
		if (prop.User != 0) && (tgbotbase.ChatID(prop.User) != prop.Chat) {
			continue
		}
		chatsToNotify = append(chatsToNotify, prop.Chat)
	}

	prevDealName, err := h.props.GetProperty("mtgsaleDealLast", 0, 0)
	if err != nil {
		panic(fmt.Sprintf("Could not get last mtgsale deal, err: %s", err))
	}

	go func() {
		data := mtgsaleDealUpdate{}
		for {
			select {
			case data = <-h.updates:
				if data.cardname == prevDealName {
					continue
				}

				prevDealName = data.cardname
				h.props.SetPropertyForUserInChat("mtgsaleDealLast", 0, 0, prevDealName)

				picFName, err := loadPicToTmp(data.picUrl, "tgbotmtg-mtgsaledeal-")
				if err != nil {
					log.Errorf("Could not load daily deal pic, err: %s", err)
					continue
				}

				text := fmt.Sprintf("Карта дня на mtgsale:\n[%s](%s)\n%s ~%s~", data.cardname, data.url, data.priceNew, data.priceOld)
				for _, chatID := range chatsToNotify {
					msg := tgbotapi.NewPhotoUpload(int64(chatID), picFName)
					msg.Caption = text
					msg.ParseMode = "MarkdownV2"
					h.OutMsgCh <- msg
				}
			}
		}
	}()

	h.cron.AddJob(time.Now(), &mtgsaleDealJob{updates: h.updates})
}

func (h *mtgSaleDealHandler) Name() string {
	return "mtgsale new deal"
}

type mtgsaleDealJob struct {
	updates chan<- mtgsaleDealUpdate
}

func (job *mtgsaleDealJob) Do(scheduledWhen time.Time, cron tgbotbase.Cron) {
	defer cron.AddJob(scheduledWhen.Add(30*time.Minute), job)

	curDeal := mtgsaleDealUpdate{}
	c := colly.NewCollector()
	c.SetRequestTimeout(20 * time.Second)
	c.OnHTML("div.cartday", func(e *colly.HTMLElement) {
		curDeal.cardname = e.ChildText(".ccart h3 a")
		curDeal.url = "https://mtgsale.ru" + e.ChildAttr(".ccart h3 a", "href")
		curDeal.picUrl = "https://mtgsale.ru" + e.ChildAttr("p.cartday a img", "src")
		curDeal.priceNew = e.ChildText(".ccart .price .new")
		curDeal.priceOld = e.ChildText(".ccart .price .old")
	})

	err := c.Visit("https://mtgsale.ru")
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Unable to visit mtgsale deal with scraper")
	}

	log.WithFields(log.Fields{
		"card":     curDeal.cardname,
		"url":      curDeal.url,
		"picUrl":   curDeal.picUrl,
		"priceNew": curDeal.priceNew,
		"priceOld": curDeal.priceOld}).Debug("scrapped mtgsale deal")

	job.updates <- curDeal
}
