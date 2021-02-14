package tester

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"
	"go.uber.org/zap"
	"io/ioutil"
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
		NodeNumber int `json:"node-number"`
	}

	Tester struct {
		central  *central.Resolver
		peers    []archive.Peer
		Sheet    *xlsx.Sheet
		logger   *zap.SugaredLogger
		centErrs <-chan error
		peerErrs <-chan error
		config   Config
	}

	Job struct {
		target  string
		execAt  time.Time
		reqType string
	}

	ResultRow struct {
		Type       string  `xlsx:"1"`
		StatusCode int     `xlsx:"2"`
		ReqFailed  bool    `xlsx:"3"`
		ReqError   string  `xlsx:"4"`
		ReqTime    float64 `xlsx:"5"`
		ReadError  string  `xlsx:"6"`
		DataSize   int     `xlsx:"7"`
	}
)

func New(ctx context.Context, config Config, centralConfig central.Config, logger *zap.SugaredLogger) (*Tester, error) {
	ts := &Tester{config: config, logger: logger}

	ts.central, ts.centErrs = ts.createCentral(ctx, centralConfig)
	ts.peers, ts.peerErrs = ts.createPeers(ctx)

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
	poolSize := ts.config.Get.PerSecond * 5
	jobs := make(chan Job, poolSize)
	results := make(chan ResultRow, poolSize)
	done := make(chan struct{})
	wg := sync.WaitGroup{}
	defer close(jobs)
	defer close(done)

	// collector thread
	go func() {
		select {
		case <-ts.centErrs:
			err := <-ts.centErrs
			ts.logger.Error(err)
		case <-ts.peerErrs:
			err := <-ts.peerErrs
			ts.logger.Error(err)
		case <-results:
			res := <-results
			row := ts.Sheet.AddRow()
			row.WriteStruct(&res, -1)
		case <-done:
			break
		case <-ctx.Done():
			break
		}
	}()

	// worker pool
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

	peers, err := ts.central.Records().AllPeers(ctx)
	if err != nil {
		return errors.Wrap(err, "querying peers")
	}

	// add store jobs to queue
	storeGap := time.Second / time.Duration(ts.config.Store.PerSecond)
	for i := time.Duration(0); i < storeDuration; i += storeGap {
		url := ts.config.Store.Targets[int(i)%len(ts.config.Store.Targets)]
		pid := peers[int(i)%len(peers)].ID
		jobs <- Job{
			target:  fmt.Sprintf("https://localhost/api/page?url=%s&peer=%d", url, pid),
			execAt:  time.Now().Add(i),
			reqType: "store",
		}
		wg.Add(1)
	}
	wg.Wait()

	snaps, err := ts.central.Records().All(ctx)
	if err != nil {
		return errors.Wrap(err, "querying snapshots")
	}

	// add get jobs to queue
	getGap := time.Second / time.Duration(ts.config.Get.PerSecond)
	for i := time.Duration(0); i < getDuration; i += getGap {
		sid := snaps[int(i)%len(snaps)].ID
		jobs <- Job{
			target:  fmt.Sprintf("https://localhost/api/page/%d", sid),
			execAt:  time.Now().Add(i),
			reqType: "get",
		}
		wg.Add(1)
	}
	wg.Wait()

	return nil
}

func getResult(start time.Time, job Job) ResultRow {
	rowVal := ResultRow{}
	resp, err := http.Get(job.target)
	rowVal.ReqTime = time.Now().Sub(start).Seconds()
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
	// clear old peers and snapshots
	_, err := ts.central.Database().ExecContext(ctx, "TRUNCATE TABLE snapshots;")
	if err != nil {
		return errors.Wrap(err, "clearing snapshots")
	}
	_, err = ts.central.Database().ExecContext(ctx, "TRUNCATE TABLE peers;")
	if err != nil {
		return errors.Wrap(err, "clearing peers")
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

func (ts *Tester) createPeers(ctx context.Context) ([]archive.Peer, <-chan error) {
	storePath := "./store"
	portStart := 25566
	// portEnd := 30000
	var peers []archive.Peer
	errs := make(chan error)

	for i := portStart; i < portStart+ts.config.NodeNumber; i++ {
		node := child.NewListener(child.Config{
			Port:      i,
			StorePath: fmt.Sprintf("%s/%d", storePath, i),
		})
		peers = append(peers, archive.Peer{
			ID:      i,
			Address: fmt.Sprintf("localhost:%d", i),
		})
		go func(port int) {
			if err := node.Listen(); err != nil {
				errs <- errors.Wrapf(err, "failed to listen on port %d", port)
			}
		}(i)
	}
	return peers, errs
}
