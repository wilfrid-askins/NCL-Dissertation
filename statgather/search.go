package main

import (
	"bytes"
	"code.sajari.com/docconv"
	"fmt"
	"io/ioutil"
	"strings"
)

func main() {
	files, err := ioutil.ReadDir("./files")
	if err != nil {
		panic(err)
	}
	countWBM := 0
	countHttp := 0
	totalFiles := len(files)
	for i, f := range files {
		data, err := ioutil.ReadFile("./files/" + f.Name())
		if err != nil {
			println(err.Error())
			continue
		}

		res, _, err := docconv.ConvertPDF(bytes.NewReader(data))
		if err != nil {
			println(err.Error())
			continue
		}
		res = strings.ToLower(res)
		countWBM += strings.Count(res, "archive.org")
		countWBM += strings.Count(res, "archive.fo")
		countWBM += strings.Count(res, "archive.is")
		countWBM += strings.Count(res, "archive.eu")
		countWBM += strings.Count(res, "archive.today")
		countWBM += strings.Count(res, "archive-it.org")
		countWBM += strings.Count(res, "perma.cc")
		countWBM += strings.Count(res, "pagefreezer.com")
		countWBM += strings.Count(res, "coralcdn.org")
		countWBM += strings.Count(res, ".nyud.net")
		countWBM += strings.Count(res, "webcache.googleusercontent.com/")
		countHttp += strings.Count(res, "http")

		fmt.Printf("%s, %d found, %d urls, %d left\n", f.Name(), countWBM, countHttp, totalFiles-i)

		// countWBM += strings.Count(res, "web.archive.org/web/")
	}

	fmt.Printf("URLs %d\n", countHttp)
	fmt.Printf("Archived %d\n", countWBM)

	if err != nil {
		panic(err)
	}
}
