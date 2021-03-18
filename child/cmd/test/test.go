package main

import (
	"context"
	"sync"
	"time"
	"wilfrid.me/netpb/pkg/netpb"
)

func main() {
	ctx, cfn := context.WithTimeout(context.Background(), time.Second*3)
	defer cfn()
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		cons, errs, err := netpb.Serve(ctx, 25565)
		if err != nil {
			panic(err.Error())
		}
		wg.Done()
	loop:
		for {
			select {
			case con := <-cons:
				go handle(ctx, con)
			case err := <-errs:
				println(err)
			case <-ctx.Done():
				break loop
			}
		}
	}()

	wg.Wait()
	con, err := netpb.Connect(ctx, "127.0.0.1:25565")
	if err != nil {
		panic(err.Error())
	}
	con.Send <- &netpb.PageRequest{
		Id: 1234,
	}

loop:
	for {
		select {
		case <-con.Rec:
			println("pong")
		case err := <-con.Errs:
			println(err.Error())
		case <-ctx.Done():
			println("done")
			break loop
		}
	}
}

func handle(ctx context.Context, con netpb.Connection) {
	println("handle")
loop:
	for {
		select {
		case <-con.Rec:
			println("ping")
			con.Send <- &netpb.PageRequest{
				Id: 1234,
			}
		case err := <-con.Errs:
			println(err.Error())
		case <-ctx.Done():
			break loop
		}
	}
}
