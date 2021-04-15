package main

import (
	"statsgather/article"
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

// Response is a response decoded from the Core API.
type Response struct {
	Articles []article.Article `json:"data"`
}

// main will take sampleSize articles from the year specified.
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
		// get next page of content
		target := getURL(apiKey, year, page, pageSize)
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			println(err.Error())
			continue
		}
		req.Header.Set("User-Agent", "Golang_Sample_Bot/1.0")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		// decode API response
		decode := Response{}
		if err := json.Unmarshal(data, &decode); err != nil {
			fmt.Println(err.Error())
			continue
		}

		for _, article := range decode.Articles {
			if len(article.DownloadURL) == 0 {
				continue
			}
			fmt.Println(article.DownloadURL)

			// put article into spreadsheet
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

	// write spreadsheet to file
	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := sheet.File.Write(writer); err != nil {
		panic(err)
	}
	writeResult(data.Bytes())
}

// writeResult will write spreadsheet data to a timestamped file.
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

// getURL returns the Core API URL with the criteria specified.
func getURL(apiKey, year string, page, pageSize int) string {
	return fmt.Sprintf("https://core.ac.uk:443/api-v2/articles/search/year:%s?page=%d&pageSize=%d&apiKey=%s", year, page, pageSize, apiKey)
}
