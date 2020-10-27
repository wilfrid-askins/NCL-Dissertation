package internal

type Config struct {
	Database string `json:"database"`
}

func LoadConfig() Config {
	return Config{
		Database: "",
	}
}
