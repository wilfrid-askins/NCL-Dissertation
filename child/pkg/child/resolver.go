package child

import (
	"go.uber.org/zap"
	"wilfrid.me/archive-child/pkg/child/file"
	"wilfrid.me/archive-child/pkg/child/server"
)

// Resolver brings together the dependencies of the child server.
type Resolver struct {
	files  *file.Repository
	server *server.Server
	logger *zap.SugaredLogger
	config Config
}

// NewResolver creates a new dependency resolver using the supplied configuration.
func NewResolver(config Config) *Resolver {
	return &Resolver{
		config: config,
	}
}

// Server returns the HTTP server singleton.
func (re *Resolver) Server() *server.Server {
	if re.server != nil {
		return re.server
	}
	re.server = server.New(re.config.Port, re.Files(), re.Logger())
	return re.server
}

// Files returns the repository to interact with the file system.
func (re *Resolver) Files() *file.Repository {
	if re.files != nil {
		return re.files
	}

	files, err := file.NewRepository(re.config.StorePath)
	if err != nil {
		re.Logger().Fatalw("failed to create file repo", "msg", err.Error())
	}

	re.files = files
	return re.files
}

// Logger returns a zap sugared logger instance.
func (re *Resolver) Logger() *zap.SugaredLogger {
	if re.logger != nil {
		return re.logger
	}
	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()
	re.logger = sugar
	return re.logger
}
