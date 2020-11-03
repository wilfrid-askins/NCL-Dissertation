package internal

import (
	"database/sql"
	"go.uber.org/zap"
	"wilfrid.me/archive-central/internal/peers"
	"wilfrid.me/archive-central/internal/record"
	"wilfrid.me/archive-central/internal/web"
)

type Resolver struct {
	db *sql.DB
	group *peers.Group
	server *web.Server
	records *record.Repository
	logger *zap.SugaredLogger
	config Config
}

func NewResolver(config Config) *Resolver {
	return &Resolver{
		config: config,
	}
}

func (re *Resolver) Database() *sql.DB {
	if re.db != nil {
		return re.db
	}
	// connect to database
	conn, err := sql.Open("mymysql", re.config.Database)
	if err != nil {
		re.Logger().Fatalw("connecting to db", "err", err)
	}
	re.db = conn
	return re.db
}

func (re *Resolver) Group() *peers.Group {
	if re.group != nil {
		return re.group
	}
	re.group = peers.NewGroup()
	return re.group
}

func (re *Resolver) WebServer() *web.Server {
	if re.server != nil {
		return re.server
	}
	re.server = web.NewServer(re.Group(), re.Logger())
	return re.server
}

func (re *Resolver) Records() *record.Repository {
	if re.records != nil {
		return re.records
	}
	records, err := record.NewRepository(re.Database())
	if err != nil {
		re.Logger().Fatalw("creating records repository", "err", err)
	}
	re.records = records
	return re.records
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
