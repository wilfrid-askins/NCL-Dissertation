package main

import (
	"RedditStats/article"
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
	apiKey     = "VKdWZtHkszLIoqe1uv75agmpEbSn0TOJ"
	pageSize   = 100
	sampleSize = 10_000
	year       = "2019"
)

type Response struct {
	Articles []article.Article `json:"data"`
}

func main() {
	needed := sampleSize
	page := 1
	f := xlsx.NewFile()
	sheet, err := f.AddSheet(fmt.Sprintf("For year %s", year))
	if err != nil {
		panic(err)
	}
	client := http.Client{}
outer:
	for {
		target := getURL(apiKey, year, page, pageSize)

		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			println(err.Error())
			continue
		}
		req.Header.Set("User-Agent", "Golang_Archive_Bot/1.0")

		resp, err := client.Do(req)
		if err != nil {
			println(err.Error())
			continue
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		decode := Response{}
		if err := json.Unmarshal(data, &decode); err != nil {
			println(err.Error())
			continue
		}

		for _, article := range decode.Articles {
			if len(article.DownloadURL) == 0 {
				continue
			}
			println(article.DownloadURL)

			row := sheet.AddRow()
			row.WriteStruct(&article, -1)
			needed--
			if needed == 0 {
				break outer
			}
		}
		fmt.Printf("%d articles left\n", needed)
		time.Sleep(time.Second * 5)
		page += 1
	}

	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := sheet.File.Write(writer); err != nil {
		panic(err)
	}
	writeResult(data.Bytes())
}

func writeResult(data []byte) {
	fileName := fmt.Sprintf("posts-%s.xlsx", time.Now().Format("2006-01-02T15-04-05"))
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

func getURL(apiKey, year string, page, pageSize int) string {
	return fmt.Sprintf("https://core.ac.uk:443/api-v2/articles/search/year:%s?page=%d&pageSize=%d&apiKey=%s", year, page, pageSize, apiKey)
}
