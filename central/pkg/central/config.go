// Package central configures and resolves the dependencies of the central archive server.

package central

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/consul/api"
)

// consulEnv is an environment variable used in a container to resolve the config server address.
const consulEnv = "CONSUL_ADDRESS"

type (
	// Config is the root of the central server configuration.
	Config struct {
		Database  Database   `json:"database"`
		Server    HTTPServer `json:"http-server"`
		Cache     Cache      `json:"cache"`
		Snapshots Snapshots  `json:"snapshots"`
	}

	// Database configures a database for the records repository to connect to.
	Database struct {
		Driver   string `json:"driver"`
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// HTTPServer defines where the central server endpoints should be available at.
	HTTPServer struct {
		Port int `json:"port"`
	}

	// Cache describes the connection details of the cache server.
	Cache struct {
		Address  string `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database int    `json:"database"`
	}

	// Snapshots defines any information related to snapshot storage.
	Snapshots struct {
		Mirrors int `json:"mirrors"`
	}
)

// DatasourceAddress returns the address for the database driver to connect to.
func (d Database) DatasourceAddress() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", d.Username, d.Password, d.Address, d.Port, d.Name)
}

// LoadConfig resolves and queries the config server, then parses and returns the result.
func LoadConfig() Config {
	var consulConf *api.Config
	if url, pres := os.LookupEnv(consulEnv); pres {
		consulConf = &api.Config{
			Address: url,
			Scheme:  "http",
		}
	} else {
		// use address 127.0.0.1:8500
		consulConf = api.DefaultConfig()
	}

	client, err := api.NewClient(consulConf)
	if err != nil {
		panic(err)
	}

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
