package central

import (
	"database/sql"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"wilfrid.me/archive-central/pkg/central/peer"
	"wilfrid.me/archive-central/pkg/central/record"
	"wilfrid.me/archive-central/pkg/central/web"
)

// Resolver puts together all the dependencies of the server central.
type Resolver struct {
	db      *sql.DB
	cache   *redis.Client
	group   peer.Group
	server  *web.Server
	records record.Repository
	logger  *zap.SugaredLogger
	config  Config
}

// NewResolver returns a new resolver for the given config.
func NewResolver(config Config) *Resolver {
	return &Resolver{
		config: config,
	}
}

// Database returns a database singleton, creating one if needed.
func (re *Resolver) Database() *sql.DB {
	if re.db != nil {
		return re.db
	}
	conn, err := sql.Open("mysql", re.config.Database.DatasourceAddress())
	if err != nil {
		re.Logger().Fatalw("connecting to db", "err", err)
	}
	re.db = conn
	re.db.SetConnMaxLifetime(time.Minute * 3)
	re.db.SetMaxOpenConns(10)
	re.db.SetMaxIdleConns(10)
	return re.db
}

// Cache returns a cache connection singleton.
func (re *Resolver) Cache() *redis.Client {
	if re.cache != nil {
		return re.cache
	}
	re.cache = redis.NewClient(&redis.Options{
		Addr:         re.config.Cache.Address,
		Username:     re.config.Cache.Username,
		Password:     re.config.Cache.Password,
		DB:           re.config.Cache.Database,
		MaxRetries:   3,
		DialTimeout:  time.Second * 10,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		MinIdleConns: 10,
	})
	return re.cache
}

// Group returns a network peer group singleton.
func (re *Resolver) Group() peer.Group {
	if re.group != nil {
		return re.group
	}
	re.group = peer.NewNetwork(re.Records(), re.Cache(), re.config.Snapshots.Mirrors, re.Logger())
	return re.group
}

// WebServer returns a server to take web requests.
func (re *Resolver) WebServer() *web.Server {
	if re.server != nil {
		return re.server
	}
	re.server = web.NewServer(re.Group(), re.Records(), re.Logger())
	return re.server
}

// Records returns a repository to interact with snapshot metadata.
func (re *Resolver) Records() record.Repository {
	if re.records != nil {
		return re.records
	}
	records, err := record.NewDatabase(re.Database())
	if err != nil {
		re.Logger().Fatalw("creating records repository", "err", err)
	}
	re.records = records
	return re.records
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
