package tester

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"
	"go.uber.org/zap"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"wilfrid.me/archive-central/pkg/central"
	"wilfrid.me/archive-child/pkg/child"
	"wilfrid.me/netpb/pkg/archive"
)

type (
	Config struct {
		Name  string `json:"name"`
		Store struct {
			Duration  string   `json:"duration"`
			PerSecond int      `json:"per-second"`
			Targets   []string `json:"targets"`
		} `json:"store"`
		Get struct {
			Duration  string `json:"duration"`
			PerSecond int    `json:"per-second"`
		} `json:"get"`
		Nodes struct {
			Number   int `json:"number"`
			Failures int `json:"failures"`
		} `json:"nodes"`
	}

	Tester struct {
		central     *central.Resolver
		peers       []archive.Peer
		peerCancels map[int]context.CancelFunc
		Sheet       *xlsx.Sheet
		logger      *zap.SugaredLogger
		centErrs    <-chan error
		peerErrs    <-chan error
		config      Config
	}

	Job struct {
		target  string
		execAt  time.Time
		reqType string
	}

	ResultRow struct {
		Type        string  `xlsx:"1"`
		Target      string  `xlsx:"2"`
		StatusCode  int     `xlsx:"3"`
		ReqFailed   bool    `xlsx:"4"`
		ReqError    string  `xlsx:"5"`
		ReqTime     float64 `xlsx:"6"`
		ReqDuration float64 `xlsx:"7"`
		ReadError   string  `xlsx:"8"`
		DataSize    int     `xlsx:"9"`
	}
)

func New(ctx context.Context, config Config, centralConfig central.Config, logger *zap.SugaredLogger) (*Tester, error) {
	ts := &Tester{config: config, logger: logger}

	ts.central, ts.centErrs = ts.createCentral(ctx, centralConfig)
	ts.peers, ts.peerCancels, ts.peerErrs = ts.createPeers(ctx)

	f := xlsx.NewFile()
	sheet, err := f.AddSheet(config.Name)
	if err != nil {
		return nil, errors.Wrap(err, "creating spreadsheet")
	}
	ts.Sheet = sheet
	return ts, nil
}

func (ts *Tester) Execute(ctx context.Context) error {
	if err := ts.initTables(ctx); err != nil {
		return errors.Wrap(err, "initiating tables")
	}
	// calculate delays
	getDuration, err := time.ParseDuration(ts.config.Get.Duration)
	if err != nil {
		return errors.Wrap(err, "parsing getDuration")
	}
	storeDuration, err := time.ParseDuration(ts.config.Store.Duration)
	if err != nil {
		return errors.Wrap(err, "parsing storeDuration")
	}

	// create worker pool
	start := time.Now()
	poolSize := ts.config.Get.PerSecond * int(getDuration.Seconds())
	jobs := make(chan Job, poolSize)
	results := make(chan ResultRow, poolSize)
	done := make(chan struct{})
	var wg sync.WaitGroup
	defer close(jobs)
	defer close(done)

	go func() {
		ts.logger.Info("running collector thread")
		for {
			select {
			case err := <-ts.centErrs:
				ts.logger.Error(err)
			case err := <-ts.peerErrs:
				ts.logger.Error(err)
			case res := <-results:
				row := ts.Sheet.AddRow()
				row.WriteStruct(&res, -1)
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	ts.logger.Info("creating worker pool")
	for i := 0; i < poolSize; i++ {
		go func() {
			for job := range jobs {
				// wait till start needed
				for time.Now().Before(job.execAt) {
				}
				results <- getResult(start, job)
				wg.Done()
			}
		}()
	}

	peers, err := ts.central.Records().GetPeers(ctx, -1)
	if err != nil {
		return errors.Wrap(err, "querying peers")
	}

	ts.logger.Info("queueing store jobs")
	storeGap := time.Second / time.Duration(ts.config.Store.PerSecond)
	totalStores := 0
	for i := time.Duration(0); i < storeDuration; i += storeGap {
		url := ts.config.Store.Targets[totalStores%len(ts.config.Store.Targets)]
		pid := peers[totalStores%len(peers)].ID
		jobs <- Job{
			target:  fmt.Sprintf("http://localhost:25565/api/store?url=%s&peer=%d", url, pid),
			execAt:  time.Now().Add(i),
			reqType: "store",
		}
		wg.Add(1)
		totalStores++
	}
	ts.logger.Info("waiting for store jobs")
	wg.Wait()

	snaps, err := ts.central.Records().All(ctx)
	if err != nil {
		return errors.Wrap(err, "querying snapshots")
	}

	// shutdown 'failed' nodes
	failures := getFailures(ts.config.Nodes.Number, ts.config.Nodes.Failures)
	for key, val := range failures {
		if !val {
			continue
		}
		cfn := ts.peerCancels[key]
		cfn()
	}

	ts.logger.Info("queueing get jobs")
	getGap := time.Second / time.Duration(ts.config.Get.PerSecond)
	totalGets := 0
	for i := time.Duration(0); i < getDuration; i += getGap {
		sid := snaps[totalGets%len(snaps)].ID
		jobs <- Job{
			target:  fmt.Sprintf("http://localhost:25565/api/page/%d", sid),
			execAt:  time.Now().Add(i),
			reqType: "get",
		}
		wg.Add(1)
		totalGets++
	}
	ts.logger.Info("waiting for get jobs")
	wg.Wait()
	ts.logger.Info("all jobs finished")

	return nil
}

func getResult(start time.Time, job Job) ResultRow {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	rowVal := ResultRow{}
	startReq := time.Now()

	rowVal.ReqTime = startReq.Sub(start).Seconds()
	resp, err := client.Get(job.target)
	rowVal.ReqDuration = time.Now().Sub(startReq).Seconds()

	rowVal.Target = job.target
	rowVal.Type = job.reqType
	rowVal.ReqFailed = err != nil
	if err != nil {
		rowVal.ReqError = err.Error()
		return rowVal
	}

	rowVal.StatusCode = resp.StatusCode
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		rowVal.ReadError = err.Error()
		return rowVal
	}
	rowVal.DataSize = len(data)
	return rowVal
}

func (ts *Tester) initTables(ctx context.Context) error {
	// clear old tables
	cmds := []string{
		"SET FOREIGN_KEY_CHECKS = 0;",
		"TRUNCATE TABLE mirrors;",
		"TRUNCATE TABLE snapshots;",
		"TRUNCATE TABLE peers;",
		"SET FOREIGN_KEY_CHECKS = 1;",
	}
	for _, cmd := range cmds {
		_, err := ts.central.Database().ExecContext(ctx, cmd)
		if err != nil {
			return errors.Wrap(err, "running db cmd")
		}
	}
	// add peers to database
	for _, p := range ts.peers {
		_, err := ts.central.Records().AddPeer(ctx, p)
		if err != nil {
			return errors.Wrap(err, "creating peer")
		}
	}
	return nil
}

func (ts *Tester) createCentral(ctx context.Context, config central.Config) (*central.Resolver, <-chan error) {
	node := central.NewResolver(config)
	ws := node.WebServer()
	port := fmt.Sprintf(":%d", config.Server.Port)
	errs := make(chan error)
	go func() {
		if err := http.ListenAndServe(port, ws.GetRouter()); err != nil {
			errs <- errors.Wrap(err, "http server listening")
		}
	}()
	return node, errs
}

func (ts *Tester) createPeers(ctx context.Context) ([]archive.Peer, map[int]context.CancelFunc, <-chan error) {
	storePath := "./store"
	portStart := 25566
	// portEnd := 30000
	var peers []archive.Peer
	peerCancels := make(map[int]context.CancelFunc)
	errs := make(chan error)

	for i := 0; i < ts.config.Nodes.Number; i++ {
		port := portStart + i
		res := child.NewResolver(child.Config{
			Port:      port,
			StorePath: fmt.Sprintf("%s/%d/", storePath, i),
		})
		peers = append(peers, archive.Peer{
			ID:      i,
			Address: fmt.Sprintf("localhost:%d", port),
		})
		nctx, cfn := context.WithCancel(ctx)
		peerCancels[i] = cfn
		go func(nctx context.Context, port int) {
			if err := res.Server().Listen(nctx); err != nil {
				errs <- errors.Wrapf(err, "failed to listen on port %d", port)
			}
		}(nctx, port)
	}
	return peers, peerCancels, errs
}

func getFailures(nodes, fails int) map[int]bool {
	failCount := 0
	failures := make(map[int]bool)
	for failCount < fails {
		ind := rand.Intn(nodes) + 1
		if failures[ind] {
			continue
		}
		failures[ind] = true
		failCount++
	}
	return failures
}
