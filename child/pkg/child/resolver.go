package child

import (
	"go.uber.org/zap"
	"wilfrid.me/archive-child/pkg/child/file"
	"wilfrid.me/archive-child/pkg/child/server"
)

type Resolver struct {
	files  *file.Repository
	server *server.Server
	logger *zap.SugaredLogger
	config Config
}

func NewResolver(config Config) *Resolver {
	return &Resolver{
		config: config,
	}
}

func (re *Resolver) Server() *server.Server {
	if re.server != nil {
		return re.server
	}
	re.server = server.New(re.config.Port, re.Files(), re.Logger())
	return re.server
}

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

func (re *Resolver) Logger() *zap.SugaredLogger {
	if re.logger != nil {
		return re.logger
	}
	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()
	re.logger = sugar
	return re.logger
}
