package bot

import (
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
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		panic(err)
	}

	return &PicCache{
		dir: baseDir,
	}
}

func (c *PicCache) Get(id, url string) (string, error) {
	fpath := path.Join(c.dir, string(id))
	_, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		return c.load(id, url)
	}
	return fpath, nil
}

func (c *PicCache) load(id, url string) (string, error) {
	log.WithFields(log.Fields{"id": id, "url": url}).Info("loading missing picture")
	resp, err := http.Get(url)
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
