package internal

type Config struct {
	Database string `json:"database"`
	Server Server `json:"server"`
}

type Server struct{
	Address string `json:"address"`
	Port int `json:"port"`
}

func LoadConfig() Config {
	return Config{
		Database: "",
		Server: Server{
			Address: "127.0.0.1",
			Port:    25565,
		},
	}
}
