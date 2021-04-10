package central

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
)

type (
	Config struct {
		Database  Database   `json:"database"`
		Server    HTTPServer `json:"http-server"`
		Cache     Cache      `json:"cache"`
		Snapshots Snapshots  `json:"snapshots"`
	}

	Database struct {
		Driver   string `json:"driver"`
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	HTTPServer struct {
		Port int `json:"port"`
	}

	Cache struct {
		Address  string `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database int    `json:"database"`
	}

	Snapshots struct {
		Mirrors int `json:"mirrors"`
	}
)

func (d Database) DatasourceAddress() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", d.Username, d.Password, d.Address, d.Port, d.Name)
}

func LoadConfig() Config {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		panic(err)
	}

	// TODO: Next steps
	// check environment variable first for consul config
	// then try default url

	kv, _, err := client.KV().Get("archive/config", &api.QueryOptions{})
	if err != nil {
		panic(err)
	}

	conf := Config{}
	if err := json.Unmarshal(kv.Value, &conf); err != nil {
		panic(err)
	}
	return conf
}
