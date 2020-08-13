package bot

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var markdownToEscape = []string{"\\", "`", "*", "_", "{", "}", "[", "]", "(", ")", "#", "+", "-", ".", "!"}

func escapeMarkdown(s string) string {
	for _, e := range markdownToEscape {
		s = strings.ReplaceAll(s, e, "\\"+e)
	}
	return s
}

func loadPicToTmp(url, prefix string) (string, error) {
	tmp, err := ioutil.TempFile("", prefix)
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(tmp, resp.Body)
	if err != nil {
		return "", err
	}

	return tmp.Name(), nil
}

func formatPrice(prefix string, p price) string {
	return fmt.Sprintf("%s %d₽ at [%s](%s)", prefix, p.Price, p.Seller, p.URL)
}
