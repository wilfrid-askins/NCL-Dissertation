package internal

import (
	"go.uber.org/zap"
	"wilfrid.me/archive-child/internal/file"
	"wilfrid.me/netpb/pkg/netpb"
)

type Resolver struct {
	files *file.Repository
	listener *netpb.Listener
	logger *zap.SugaredLogger
	config Config
}

func NewResolver(config Config) *Resolver {
	return &Resolver{
		config: config,
	}
}

func (re *Resolver) Files() *file.Repository {
	if re.files != nil {
		return re.files
	}

	files, err := file.NewRepository()
	if err != nil {
		re.Logger().Fatalw("failed to create file repo", "msg", err.Error())
	}

	re.files = files
	return re.files
}

func (re *Resolver) Listener() *netpb.Listener {
	if re.listener != nil {
		return re.listener
	}
	re.listener = netpb.NewListener(re.Logger())
	return re.listener
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
