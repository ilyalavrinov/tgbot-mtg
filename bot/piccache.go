package bot

import (
	"os"
	"path"
)

type CardID string

type PicCache struct {
	dir string
}

func NewPicCache(baseDir string) *PicCache {
	return &PicCache{
		dir: baseDir,
	}
}

func (c *PicCache) Get(id CardID) string {
	fpath := path.Join(c.dir, string(id))
	_, err := os.Stat(fpath)
	if err == os.ErrNotExist {
		return c.load(id)
	}
	return fpath
}

func (c *PicCache) load(id CardID) string {
	return path.Join(c.dir, string(id))
}
