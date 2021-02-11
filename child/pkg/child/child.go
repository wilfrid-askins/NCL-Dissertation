package child

import (
	"wilfrid.me/archive-child/internal"
	"wilfrid.me/archive-child/internal/peer"
	"wilfrid.me/netpb/pkg/netpb"
)

func NewListener(config Config) *netpb.Listener {
	resolver := internal.NewResolver(config)
	_ = peer.New(resolver.Listener().Events, resolver.Files(), resolver.Logger())
	return resolver.Listener()
}
