module wilfrid.me/archive-test

go 1.16

replace wilfrid.me/archive-central v0.0.0 => ../central/

replace wilfrid.me/archive-child v0.0.0 => ../child/

require (
	github.com/gobuffalo/packr/v2 v2.8.1
	github.com/pkg/errors v0.9.1
	github.com/tealeg/xlsx v1.0.5
	go.uber.org/zap v1.16.0
	wilfrid.me/archive-central v0.0.0
	wilfrid.me/archive-child v0.0.0
)
