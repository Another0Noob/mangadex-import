package config

import (
	"fmt"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"gopkg.in/ini.v1"
)

func LoadAuth(path string) (mangadexapi.AuthForm, error) {
	var m mangadexapi.AuthForm
	cfg, err := ini.Load(path)
	if err != nil {
		return m, fmt.Errorf("load auth config %s: %w", path, err)
	}
	sec := cfg.Section("mangadex")
	m.Username = sec.Key("username").String()
	m.Password = sec.Key("password").String()
	m.ClientID = sec.Key("client_id").String()
	m.ClientSecret = sec.Key("client_secret").String()
	return m, nil
}
