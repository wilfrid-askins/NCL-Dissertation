module wilfrid.me/archive-test

go 1.15

replace wilfrid.me/archive-child v0.0.0 => ../child/
replace wilfrid.me/netpb v0.0.0 => ../netpb/

require (
	github.com/gobuffalo/packr/v2 v2.8.1 // indirect
	github.com/tealeg/xlsx v1.0.5 // indirect
	wilfrid.me/archive-child v0.0.0
)
