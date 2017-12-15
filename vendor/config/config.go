package config

import (
	"appinfo"
	"database"
	"server"
	"encoding/json"
)

type Configuration struct {
	Database database.Info
	Server   server.Server
	AppInfo  appinfo.AppInfo
}

func (c *Configuration) ParseJSON(b []byte) error {
	return json.Unmarshal(b, &c)
}