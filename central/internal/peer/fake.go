package peer

import "context"

type FakeNetwork struct {
}

func (fn *FakeNetwork) Get(ctx context.Context, id int) ([]byte, error) {

}

func (fn *FakeNetwork) Store(ctx context.Context, peerID int, url string) (int, error) {

}
