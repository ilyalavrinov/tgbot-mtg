package price

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
	log "github.com/sirupsen/logrus"
)

func MtgSale(cardname string) int {
	var bestPrice int = math.MaxInt16
	c := colly.NewCollector()
	c.OnHTML(".ctclass", func(e *colly.HTMLElement) {
		name1 := strings.ToLower(e.ChildText(".tnamec"))
		name2 := strings.ToLower(e.ChildText(".smallfont"))
		cardname := strings.ToLower(cardname)
		log.WithFields(log.Fields{"name1": name1, "name2": name2, "cardname": cardname}).Debug("parsing mtgsale card")
		if name1 == cardname || name2 == cardname {
			p := e.ChildText(".pprice")
			p = strings.Trim(p, " ₽")
			pVal, err := strconv.Atoi(p)
			if err != nil {
				log.WithFields(log.Fields{"card": cardname, "price": p, "err": err}).Error("Price cannot be parsed")
				return
			}
			/*
				foil := false
				if e.ChildText(".foil") != "" {
					foil = true
				}
				count := e.ChildText(".colvo")
				count = strings.Trim(count, " шт.")
				countVal, err := strconv.Atoi(count)
				if err != nil {
					log.WithFields(log.Fields{"card": cardname, "count": count, "err": err}).Error("Count cannot be parsed")
					return
				}
			*/
			if pVal < bestPrice {
				bestPrice = pVal
			}
		}
	})

	addr := MtgSaleURL(cardname)
	err := c.Visit(addr)
	if err != nil {
		log.WithFields(log.Fields{"url": addr, "err": err}).Error("Unable to visit with scraper")
	}
	return bestPrice
}

func MtgSaleURL(cardname string) string {
	return fmt.Sprintf("https://mtgsale.ru/home/search-results?Name=%s&Lang=Any&Type=Any&Color=Any&Rarity=Any", url.PathEscape(cardname))
}
