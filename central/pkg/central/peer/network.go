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
	"golang.org/x/sync/singleflight"
	"io/ioutil"
	"net/http"
	"time"
	"wilfrid.me/archive-central/pkg/central/record"
	"wilfrid.me/netpb/pkg/archive"
	"wilfrid.me/netpb/pkg/netpb"
)

const (
	cacheTime    = time.Hour
	mirror       = true
	mirrorFactor = 3
	searchFactor = 3
	mirrorTime   = time.Minute * 2
)

type (
	Network struct {
		records      record.Repository
		cache        *redis.Client
		transfers    *netpb.Transfer
		pool         *netpb.Pool
		logger       *zap.SugaredLogger
		requestGroup singleflight.Group
	}
)

func NewNetwork(records record.Repository, cache *redis.Client, logger *zap.SugaredLogger) *Network {
	return &Network{
		records:   records,
		cache:     cache,
		transfers: netpb.NewTransfer(logger),
		pool:      netpb.NewPool(),
		logger:    logger,
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
	mirrors, err := nw.records.GetMirrors(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting mirror records")
	}
	// make array of peers to query
	var peers []archive.Peer
	peers = append(peers, snap.Peer)
	peers = append(peers, mirrors...)

	data, err, _ := nw.requestGroup.Do(fmt.Sprintf("%s-%d", snap.Peer.Address, id), func() (interface{}, error) {
		data, err := nw.getWithFallback(ctx, id, peers)
		return data, err
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting from peer")
		// TODO try mirrors if not found
	}
	bdata := (data).([]byte)

	if err := nw.cache.Set(ctx, key, bdata, cacheTime).Err(); err != nil {
		nw.logger.Errorw("failed to set cache", "msg", err.Error())
	}

	return bdata, nil
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
		Sum:  checkSum,
		URL:  url,
		Time: time.Now(),
		Peer: archive.Peer{
			ID: peerID,
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "creating new page record")
	}

	if mirror {
		go func() {
			nctx, cfn := context.WithTimeout(context.Background(), mirrorTime)
			defer cfn()
			if err := nw.sendToMirrors(nctx, int(recID), peerID, comp.Bytes()); err != nil {
				nw.logger.Errorw("sending data to mirrors", "id", recID, "err", err.Error())
			}
		}()
	}

	//con, cfn, err := nw.pool.Connect(peer.Address)
	//if err != nil {
	//	return 0, errors.Wrap(err, "connecting to peer")
	//}
	//defer cfn()
	con, err := netpb.Connect(ctx, peer.Address)
	if err != nil {
		return 0, errors.Wrap(err, "connecting to peer")
	}
	defer con.Close()

	netpb.SendPage(recID, comp.Bytes(), con.Send)

	select {
	case err := <-con.Errs:
		return 0, errors.Wrap(err, "connection error")
	default:
		return int(recID), nil
	}
}

func (nw *Network) sendToMirrors(ctx context.Context, recID, exID int, data []byte) error {
	peers, err := nw.records.GetPeers(ctx, mirrorFactor*searchFactor)
	if err != nil {
		return errors.Wrap(err, "getting list of peers")
	}
	needed := mirrorFactor
	for needed > 0 && len(peers) > 0 {
		peer := peers[0]
		peers = peers[1:]
		if peer.ID == exID {
			continue
		}

		con, err := netpb.Connect(ctx, peer.Address)
		if err != nil {
			return errors.Wrap(err, "connecting to peer")
		}
		netpb.SendPage(int64(recID), data, con.Send)

		select {
		case err := <-con.Errs:
			nw.logger.Errorw("transferring to mirror", "msg", err.Error())
			continue
		default:
			if _, err := nw.records.AddMirror(ctx, peer.ID, recID); err != nil {
				nw.logger.Errorw("adding mirror record", "msg", err.Error())
				continue
			}
			needed--
		}
		con.Close()
	}
	if needed > 0 {
		return errors.New("failed to find enough available peers")
	}
	return nil
}

func (nw *Network) getWithFallback(ctx context.Context, id int, peers []archive.Peer) ([]byte, error) {
	for _, peer := range peers {
		data, err := nw.getFromPeer(ctx, id, peer.Address)
		if err != nil {
			nw.logger.Error(err)
			continue
		}
		return data, nil
	}
	return nil, errors.New("none available")
}

func (nw *Network) getFromPeer(ctx context.Context, id int, address string) ([]byte, error) {
	var notFound bool
	var result []byte

	//con, cfn, err := nw.pool.Connect(address)
	//if err != nil {
	//	return nil, errors.Wrap(err, "connecting to peer")
	//}
	//defer cfn()
	con, err := netpb.Connect(ctx, address)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to peer")
	}
	defer con.Close()

	con.Send <- &netpb.PageRequest{
		Id: int64(id),
	}

loop:
	for {
		select {
		case p := <-con.Rec:
			switch p.(type) {
			case *netpb.PageTransfer:
				trans := p.(*netpb.PageTransfer)
				if !trans.Found {
					nw.logger.Error("file not found")
					continue
				}
				nw.transfers.New(trans)
			case *netpb.PageChunk:
				chunk := p.(*netpb.PageChunk)
				done, data := nw.transfers.Push(chunk)
				if !done {
					continue
				}
				result = data
				break loop
			default:
				nw.logger.Error("unknown packet type received")
			}
		case err := <-con.Errs:
			nw.logger.Errorw("connection returned error", "err", err.Error())
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if notFound {
		return nil, errors.New("not found")
	}

	// TODO test checksum here

	return result, nil
}
