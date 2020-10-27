package internal

import (
	"database/sql"
	"wilfrid.me/archive-central/internal/record"
	"go.uber.org/zap"
)

type Resolver struct {
	db *sql.DB
	records *record.Repository
	// listener
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

func (re *Resolver) Records() *record.Repository {
	if re.records != nil {
		return re.records
	}
	re.records = record.NewRepository(re.Database())
	return re.records
}

func (re *Resolver) Listener() {

}

func (re *Resolver) Logger() *zap.SugaredLogger {
	if re.logger != nil {
		return re.logger
	}
	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()
	re.logger = sugar
}
