package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// セーブデータは %APPDATA%\NovelGames\<ゲーム名>_<ID>\ に1キー1ファイルで保存します。

var unsafeChars = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)
var keyChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

func folderName(title, id string) string {
	name := strings.TrimSpace(unsafeChars.ReplaceAllString(title, "_"))
	name = strings.TrimRight(name, ". ")
	if r := []rune(name); len(r) > 40 {
		name = string(r[:40])
	}
	if name == "" {
		name = "game"
	}
	id = keyChars.ReplaceAllString(id, "")
	if id != "" {
		name += "_" + id
	}
	return name
}

type store struct{ dir string }

func newStore(base, title, id string) *store {
	return &store{dir: filepath.Join(base, "NovelGames", folderName(title, id))}
}

func (s *store) path(key string) string {
	k := keyChars.ReplaceAllString(key, "_")
	if k == "" {
		k = "_"
	}
	return filepath.Join(s.dir, k+".json")
}

func (s *store) Load(key string) (string, error) {
	b, err := os.ReadFile(s.path(key))
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(b), err
}

func (s *store) Save(key, value string) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	p := s.path(key)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(value), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s *store) Delete(key string) error {
	err := os.Remove(s.path(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
