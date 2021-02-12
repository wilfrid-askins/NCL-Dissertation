package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"net/http"
	"os"
	"strings"
	"time"
	"wilfrid.me/archive-central/pkg/central"
	"wilfrid.me/archive-child/pkg/child"
	"wilfrid.me/netpb/pkg/archive"
)

const root = "./tests"
const output = "./output"

var tests = []string{
	"test1",
}

type (
	Params struct {
		Name          string `json:"name"`
		Target        string `json:"target"`
		RequestNumber int    `json:"request-number"`
		Duration      string `json:"duration"`
		NodeNumber    int    `json:"node-number"`
	}

	ResultRow struct {
		StatusCode int     `xlsx:"1"`
		ReqFailed  bool    `xlsx:"2"`
		ReqError   string  `xlsx:"3"`
		ReqTime    float64 `xlsx:"4"`
		ReadError  string  `xlsx:"5"`
		DataSize   int     `xlsx:"6"`
	}
)

func main() {
	testName := pickTest()
	box := packr.New(testName, root+"/"+testName)

	peers := createPeers()
	// load config and create resolver
	// move this later
	config := central.Config{}
	configData, err := box.Find("central.json")
	if err != nil {
		panic(err.Error())
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		panic(err.Error())
	}
	resolver := central.NewResolver(config)
	ws := resolver.WebServer()
	port := fmt.Sprintf(":%d", config.Server.Port)
	go func() {
		if err := http.ListenAndServe(port, ws.GetRouter()); err != nil {
			fmt.Println(err.Error())
		}
	}()
	// add peers to database
	for _, p := range peers {
		_, err := resolver.Records().AddPeer(context.Background(), p)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	/*
		// load parameters
		params := Params{}
		paramsData, err := box.Find("params.json")
		if err := json.Unmarshal(paramsData, &params); err != nil {
			panic(err)
		}

		// create sheet
		f := xlsx.NewFile()
		sheet, err := f.AddSheet(params.Name)
		if err != nil {
			panic(err)
		}

		// calculate delays
		duration, err := time.ParseDuration(params.Duration)
		if err != nil {
			panic(err)
		}
		perRequest := duration / time.Duration(params.RequestNumber)

		// run http requests at ip address
		// record responses
		start := time.Now()
		for i := 0; i < params.RequestNumber; i++ {
			rowVal := ResultRow{}
			resp, err := http.Get(params.Target)
			rowVal.ReqTime = time.Now().Sub(start).Seconds()
			rowVal.ReqFailed = err != nil
			if err != nil {
				rowVal.ReqError = err.Error()
				continue
			}

			rowVal.StatusCode = resp.StatusCode
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				rowVal.ReadError = err.Error()
				continue
			}
			rowVal.DataSize = len(data)

			row := sheet.AddRow()
			row.WriteStruct(&rowVal, -1)
			time.Sleep(perRequest)
		}

		// write sheet to file
		var data bytes.Buffer
		writer := bufio.NewWriter(&data)
		if err := f.Write(writer); err != nil {
			panic(err)
		}
		writeResult(testName, data.Bytes())
	*/
}

func createPeers() []archive.Peer {
	storePath := "./store"
	portStart := 25566
	// portEnd := 30000
	nodeNum := 500
	var peers []archive.Peer

	for i := portStart; i < portStart+nodeNum; i++ {
		node := child.NewListener(child.Config{
			Port:      i,
			StorePath: fmt.Sprintf("%s/%d", storePath, i),
		})
		peers = append(peers, archive.Peer{
			ID:      i,
			Address: fmt.Sprintf("localhost:%d", i),
		})
		go func(port int) {
			if err := node.Listen(); err != nil {
				fmt.Printf("failed to listen on port %d, %s", port, err.Error())
			}
		}(i)
	}
	return peers
}

func writeResult(testName string, data []byte) {
	fileName := fmt.Sprintf("%s-%s.xlsx", testName, time.Now().Format("2006-01-02T15:04:05"))
	fo, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()
	_, err = fo.Write(data)
	if err != nil {
		panic(err)
	}
}

func pickTest() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter test to run:")
	for _, name := range tests {
		fmt.Println(name)
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	text = strings.ToLower(text)
	text = strings.Trim(text, "\r\n ")
	if !contains(tests, text) {
		fmt.Println("Error test not found")
		return pickTest()
	}
	return text
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
