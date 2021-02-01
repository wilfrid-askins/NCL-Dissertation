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
	Group struct {
		records   *record.Repository
		cache     *redis.Client
		logger    *zap.SugaredLogger
		transfers *netpb.Transfer
	}
)

func NewGroup(records *record.Repository, cache *redis.Client, logger *zap.SugaredLogger) *Group {
	return &Group{
		records:   records,
		cache:     cache,
		logger:    logger,
		transfers: netpb.NewTransfer(logger),
	}
}

func (gr *Group) Get(ctx context.Context, id int) ([]byte, error) {
	// check cache
	key := fmt.Sprintf("snapshot-%d", id)
	cached, err := gr.cache.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		gr.logger.Errorw("failed to query cache", "msg", err.Error())
	}
	if cached != nil {
		return cached, nil
	}

	// find peer
	snap, err := gr.records.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting snapshot record")
	}

	data, err := gr.GetFromPeer(id, snap.Origin)
	if err != nil {
		return nil, errors.Wrap(err, "getting from peer")
	}

	// TODO try mirrors if not found

	if err := gr.cache.Set(ctx, key, data, cacheTime).Err(); err != nil {
		gr.logger.Errorw("failed to set cache", "msg", err.Error())
	}

	return data, nil
}

func (gr *Group) Store(ctx context.Context, peerID int, url string) (int, error) {
	// find peer to store in
	peer, err := gr.records.GetPeer(ctx, peerID)
	if err != nil {
		return 0, errors.Wrap(err, "getting peer record")
	}

	// download url to memory
	// get checksum of bytes
	// store record in db
	// get id of record
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

	recID, err := gr.records.Add(ctx, archive.Snapshot{
		Sum:    checkSum,
		URL:    url,
		Time:   time.Now(),
		Origin: peer.Address,
	})
	if err != nil {
		return 0, errors.Wrap(err, "creating new page record")
	}

	cl := netpb.NewClient(peer.Address, gr.logger)

	go func() {
		if err := cl.Connect(); err != nil {
			gr.logger.Errorw("connecting to peer", "err", err)
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

func (gr *Group) GetFromPeer(id int, address string) ([]byte, error) {
	var done bool
	var result []byte
	cl := netpb.NewClient(address, gr.logger)

	go func() {
		if err := cl.Connect(); err != nil {
			gr.logger.Errorw("connecting to peer", "err", err)
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
		gr.transfers.New(req)
	})

	cl.Events.On(netpb.ID(&netpb.PageChunk{}), func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		done, result = gr.transfers.Push(chunk)
	})

	if err := cl.Send(&netpb.PageRequest{
		Id: int64(id),
	}); err != nil {
		gr.logger.Errorw("sending page request", "err", err)
	}

	for !done {
	}

	// TODO test checksum here

	return result, nil
}
