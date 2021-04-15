package main

import (
	"bytes"
	"code.sajari.com/docconv"
	"fmt"
	"io/ioutil"
	"strings"
)

// main will search all articles in the ./files directory for occurrences of links.
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
			fmt.Println(err.Error())
			continue
		}

		// get text from PDF
		res, _, err := docconv.ConvertPDF(bytes.NewReader(data))
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		// search text for keywords
		res = strings.ToLower(res)
		countWBM += strings.Count(res, "web.archive.org/web/")
		countHttp += strings.Count(res, "http://")
		countHttp += strings.Count(res, "https://")

		fmt.Printf("%s, %d found, %d urls, %d left\n", f.Name(), countWBM, countHttp, totalFiles-i)
	}

	fmt.Printf("URLs %d\n", countHttp)
	fmt.Printf("Archived %d\n", countWBM)
}
