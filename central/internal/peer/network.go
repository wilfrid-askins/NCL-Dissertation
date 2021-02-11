package peer

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
	"wilfrid.me/archive-central/internal/record"
	"wilfrid.me/netpb/pkg/archive"
	"wilfrid.me/netpb/pkg/netpb"
)

const cacheTime = time.Hour

type (
	Network struct {
		records   record.Repository
		cache     *redis.Client
		logger    *zap.SugaredLogger
		transfers *netpb.Transfer
	}
)

func NewNetwork(records record.Repository, cache *redis.Client, logger *zap.SugaredLogger) *Network {
	return &Network{
		records:   records,
		cache:     cache,
		logger:    logger,
		transfers: netpb.NewTransfer(logger),
	}
}

func (nw *Network) Get(ctx context.Context, id int) ([]byte, error) {
	// check cache
	key := fmt.Sprintf("snapshot-%d", id)
	cached, err := nw.cache.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		nw.logger.Errorw("failed to query cache", "msg", err.Error())
	}
	if err != redis.Nil {
		return cached, nil
	}

	// find peer
	snap, err := nw.records.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting snapshot record")
	}

	data, err := nw.getFromPeer(id, snap.Origin)
	if err != nil {
		return nil, errors.Wrap(err, "getting from peer")
	}

	// TODO try mirrors if not found

	if err := nw.cache.Set(ctx, key, data, cacheTime).Err(); err != nil {
		nw.logger.Errorw("failed to set cache", "msg", err.Error())
	}

	return data, nil
}

func (nw *Network) Store(ctx context.Context, peerID int, url string) (int, error) {
	// find peer to store in
	peer, err := nw.records.GetPeer(ctx, peerID)
	if err != nil {
		return 0, errors.Wrap(err, "getting peer record")
	}

	// download url to memory
	resp, err := http.Get(url)
	if err != nil {
		return 0, errors.Wrap(err, "downloading url")
	}
	pageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Wrap(err, "reading url data")
	}

	// compress page data
	var comp bytes.Buffer
	zw := gzip.NewWriter(&comp)
	if _, err := zw.Write(pageData); err != nil {
		return 0, errors.Wrap(err, "compressing page data")
	}
	if err := zw.Close(); err != nil {
		return 0, errors.Wrap(err, "closing compression writer")
	}
	data := comp.Bytes()

	// calculate hash of page data
	hash := sha512.New512_256()
	if _, err := hash.Write(data); err != nil {
		return 0, errors.Wrap(err, "writing hash")
	}
	checkSum := fmt.Sprintf("%x", hash.Sum(nil))

	// add page record
	recID, err := nw.records.Add(ctx, archive.Snapshot{
		Sum:    checkSum,
		URL:    url,
		Time:   time.Now(),
		Origin: peer.Address,
	})
	if err != nil {
		return 0, errors.Wrap(err, "creating new page record")
	}

	cl := netpb.NewClient(peer.Address, nw.logger)

	go func() {
		if err := cl.Connect(); err != nil {
			nw.logger.Errorw("connecting to peer", "err", err)
		}
	}()

	for !cl.Connected {
	}

	// tell peer to store this page
	if err := cl.Send(&netpb.PageTransfer{
		Id:        recID,
		ChunkSize: int64(len(data)),
		ChunkNum:  1,
		TotalSize: int64(len(data)),
	}); err != nil {
		return 0, errors.Wrap(err, "page transfer msg")
	}
	if err := cl.Send(&netpb.PageChunk{
		Id:    recID,
		Chunk: 0,
		Data:  data,
	}); err != nil {
		return 0, errors.Wrap(err, "page chunk msg")
	}

	return int(recID), nil
}

func (nw *Network) getFromPeer(id int, address string) ([]byte, error) {
	var done bool
	var result []byte
	cl := netpb.NewClient(address, nw.logger)

	go func() {
		if err := cl.Connect(); err != nil {
			nw.logger.Errorw("connecting to peer", "err", err)
		}
	}()

	for !cl.Connected {
	}

	cl.Events.On(netpb.ID(&netpb.PageTransfer{}), func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageTransfer)
		if !req.Found {
			// TODO not found
			return
		}
		nw.transfers.New(req)
	})

	cl.Events.On(netpb.ID(&netpb.PageChunk{}), func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		done, result = nw.transfers.Push(chunk)
	})

	if err := cl.Send(&netpb.PageRequest{
		Id: int64(id),
	}); err != nil {
		nw.logger.Errorw("sending page request", "err", err)
	}

	for !done {
	}

	// TODO test checksum here

	return result, nil
}
