module wilfrid.me/archive-child

go 1.15

replace wilfrid.me/netpb v0.0.0 => ../netpb/

require (
	github.com/go-chi/chi v1.5.4
	github.com/golang/protobuf v1.4.3
	github.com/pkg/errors v0.8.1
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	wilfrid.me/netpb v0.0.0
)
