package bot

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

type PicCache struct {
	dir string
}

func NewPicCache(baseDir string) *PicCache {
	return &PicCache{
		dir: baseDir,
	}
}

func (c *PicCache) Get(id string) (string, error) {
	fpath := path.Join(c.dir, string(id))
	_, err := os.Stat(fpath)
	if err == os.ErrNotExist {
		return c.load(id)
	}
	return fpath, nil
}

func (c *PicCache) load(id string) (string, error) {
	log.WithFields(log.Fields{"id": id}).Info("loading missing picture")
	resp, err := http.Get(picURL(id))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fpath := path.Join(c.dir, string(id))
	out, err := os.Create(fpath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return fpath, nil
}

func picURL(id string) string {
	return fmt.Sprintf("https://img.scryfall.com/cards/border_crop/front/1/2/%s.jpg", id)
}
