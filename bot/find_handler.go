package bot

import (
	"github.com/admirallarimda/tgbotbase"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type findHandler struct {
	tgbotbase.BaseHandler

	cache *PicCache
}

var _ tgbotbase.IncomingMessageHandler = &findHandler{}

func NewFindHandler(cache *PicCache) tgbotbase.IncomingMessageHandler {
	h := findHandler{
		cache: cache,
	}
	return &h
}

func (h *findHandler) Init(outMsgCh chan<- tgbotapi.Chattable, srvCh chan<- tgbotbase.ServiceMsg) tgbotbase.HandlerTrigger {
	h.OutMsgCh = outMsgCh
	return tgbotbase.NewHandlerTrigger(nil, []string{"find"})
}

func (h *findHandler) HandleOne(msg tgbotapi.Message) {

}

func (h *findHandler) Name() string {
	return "Find Handler"
}
