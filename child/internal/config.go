package internal

type Config struct {
	Port int
}

func LoadConfig() Config {
	return Config{
		Port: 25565,
	}
}
