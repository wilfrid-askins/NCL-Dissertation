package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tealeg/xlsx"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	url  = "https://www.reddit.com/domain/web.archive.org/new.json?limit=100"
	week = time.Hour * 24 * 7
	// until = week
	until = week * 4 * 12 * 5
)

type Post struct {
	Name    string  `json:"name" xlsx:"1"`
	Title   string  `json:"title" xlsx:"2"`
	Created float64 `json:"created_utc" xlsx:"3"`
	Domain  string  `json:"domain" xlsx:"4"`
}

type Child struct {
	Kind string `json:"kind"`
	Data Post   `json:"data"`
}

type Response struct {
	Data struct {
		Children []Child `json:"children"`
	} `json:"data"`
}

func main() {
	f := xlsx.NewFile()
	sheet, err := f.AddSheet("Posts")
	if err != nil {
		panic(err)
	}
	client := http.Client{}
	endTime := time.Now().Add(-until)
	var currentPage string
timer:
	for {
		target := url
		if currentPage != "" {
			target = fmt.Sprintf("%s&after=%s", url, currentPage)
		}
		println(target)

		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("User-Agent", "Golang_Archive_Bot/1.0")

		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		decode := Response{}
		if err := json.Unmarshal(data, &decode); err != nil {
			panic(err)
		}
		children := decode.Data.Children

		counter := 0
		for _, child := range children {
			if child.Data.Created < float64(endTime.Unix()) {
				break timer
			}

			row := sheet.AddRow()
			row.WriteStruct(&child.Data, -1)
			counter++
		}
		fmt.Printf("processed %d posts at %s\n", counter, currentPage)

		last := children[len(children)-1]
		currentPage = last.Data.Name

		time.Sleep(time.Second * 2)
	}

	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := sheet.File.Write(writer); err != nil {
		panic(err)
	}
	writeResult(data.Bytes())
}

func writeResult(data []byte) {
	fileName := fmt.Sprintf("posts-%s.xlsx", time.Now().Format("2006-01-02T15:04:05"))
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
