package tester

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tealeg/xlsx"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
	"wilfrid.me/archive-central/pkg/central"
	"wilfrid.me/archive-child/pkg/child"
	"wilfrid.me/netpb/pkg/archive"
)

type (
	Config struct {
		Name          string `json:"name"`
		Target        string `json:"target"`
		RequestNumber int    `json:"request-number"`
		Duration      string `json:"duration"`
		NodeNumber    int    `json:"node-number"`
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

	ResultRow struct {
		StatusCode int     `xlsx:"1"`
		ReqFailed  bool    `xlsx:"2"`
		ReqError   string  `xlsx:"3"`
		ReqTime    float64 `xlsx:"4"`
		ReadError  string  `xlsx:"5"`
		DataSize   int     `xlsx:"6"`
	}
)

func New(config Config, centralConfig central.Config, logger *zap.SugaredLogger) (*Tester, error) {
	ts := &Tester{config: config, logger: logger}

	ts.central, ts.centErrs = ts.createCentral(ctx, centralConfig)
	ts.peers, ts.peerErrs = ts.createPeers(ctx)

	f := xlsx.NewFile()
	sheet, err := f.AddSheet(config.Name)
	if err != nil {
		return nil, errors.Wrap(err, "creating spreadsheet")
	}
	ts.Sheet = sheet
}

func (ts *Tester) Execute(ctx context.Context) error {
	if err := ts.initTables(ctx); err != nil {
		return errors.Wrap(err, "initiating tables")
	}
	// calculate delays
	duration, err := time.ParseDuration(ts.config.Duration)
	if err != nil {
		return errors.Wrap(err, "parsing duration")
	}
	perRequest := duration / time.Duration(ts.config.RequestNumber)

	// run http requests at ip address
	// record responses
	// TODO: update this function
	// needs to run multiple requests per second
	// need rw mutex on sheet
	go func() {
		start := time.Now()
		for i := 0; i < ts.config.RequestNumber; i++ {
			rowVal := ResultRow{}
			resp, err := http.Get(ts.config.Target)
			rowVal.ReqTime = time.Now().Sub(start).Seconds()
			rowVal.ReqFailed = err != nil
			if err != nil {
				rowVal.ReqError = err.Error()
				continue
			}

			rowVal.StatusCode = resp.StatusCode
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				rowVal.ReadError = err.Error()
				continue
			}
			rowVal.DataSize = len(data)

			ts.addResult(rowVal)
			time.Sleep(perRequest)
		}
	}()

	select {
	case <-ts.centErrs:
		err := <-ts.centErrs
		ts.logger.Error(err)
	case <-ts.peerErrs:
		err := <-ts.peerErrs
		ts.logger.Error(err)
	case <-ctx.Done():
		return nil
	}
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
	nodeNum := 20
	var peers []archive.Peer
	errs := make(chan error)

	for i := portStart; i < portStart+nodeNum; i++ {
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

func (ts *Tester) addResult(result ResultRow) {
	row := ts.Sheet.AddRow()
	row.WriteStruct(&result, -1)
}
