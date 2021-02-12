module wilfrid.me/archive-test

go 1.16

replace wilfrid.me/archive-central v0.0.0 => ../central/

replace wilfrid.me/archive-child v0.0.0 => ../child/

replace wilfrid.me/netpb v0.0.0 => ../netpb/

require (
	github.com/gobuffalo/packr/v2 v2.8.1 // indirect
	github.com/tealeg/xlsx v1.0.5 // indirect
	golang.org/dl v0.0.0-20210204224843-1557c60ec592 // indirect
	wilfrid.me/archive-central v0.0.0
	wilfrid.me/archive-child v0.0.0
)
