package main

import (
	"RedditStats/article"
	"fmt"
	"github.com/tealeg/xlsx"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	fileName  = "posts-2021-03-27T19-24-01.xlsx"
	sheetName = "For year 2012"
)

func main() {
	page := 1
	f, err := xlsx.OpenFile(fileName)
	if err != nil {
		panic(err)
	}
	sheet := f.Sheet[sheetName]
	client := http.Client{}

	for _, row := range sheet.Rows {
		var article article.Article
		if err := row.ReadStruct(&article); err != nil {
			panic(err)
		}
		println(article.DownloadURL)

		//if article.CountWBM != 0 || article.CountGoogle != 0 {
		//	continue
		//}

		pdfFileName := fmt.Sprintf("./files/%s.pdf", article.ID)
		if fileExists(pdfFileName) {
			continue
		}

		target, works := convertURL(article.DownloadURL)
		if !works {
			continue
		}

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
			println(err.Error())
			continue
		}

		fo, err := os.Create(pdfFileName)
		if err != nil {
			panic(err)
		}
		_, err = fo.Write(data)
		if err != nil {
			panic(err)
		}
		if err := fo.Close(); err != nil {
			panic(err)
		}

		//res, _, err := docconv.ConvertPDF(bytes.NewReader(data))
		//if err != nil {
		//	log.Fatal(err)
		//}
		//article.CountGoogle = strings.Count(res, "webcache.googleusercontent.com/")
		//article.CountWBM = strings.Count(res, "web.archive.org/web/")
		//row.WriteStruct(&article, -1)

		fmt.Printf("%d downloaded\n", page)
		time.Sleep(time.Second * 1)
		page += 1
	}

	//var data bytes.Buffer
	//writer := bufio.NewWriter(&data)
	//if err := sheet.File.Write(writer); err != nil {
	//	panic(err)
	//}
	//writeFilledResult(data.Bytes())
}

func convertURL(url string) (string, bool) {
	if strings.Contains(url, "arxiv.org/abs/") {
		parts := strings.Split(url, "/")
		id := parts[len(parts)-1]
		return fmt.Sprintf("https://arxiv.org/pdf/%s.pdf", id), true
	}
	if strings.Contains(url, "core.ac.uk/download/") {
		return url, true
	}
	return "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
