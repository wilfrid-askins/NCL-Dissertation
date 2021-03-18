package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"
	"wilfrid.me/archive-central/pkg/central"
	"wilfrid.me/archive-test/internal/tester"
)

const root = "./tests"

var tests = []string{
	"test1",
	"test2",
}

func main() {
	ctx, cfn := context.WithTimeout(context.Background(), time.Minute*10)
	defer cfn()
	prodLog, _ := zap.NewProduction()
	logger := prodLog.Sugar()

	testName := pickTest()
	box := packr.New(testName, root+"/"+testName)

	// load config and create resolver
	// move this later
	config := central.Config{}
	configData, err := box.Find("central.json")
	if err != nil {
		logger.Fatal(err.Error())
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		logger.Fatal(err.Error())
	}

	// load parameters
	params := tester.Config{}
	paramsData, err := box.Find("test.json")
	if err := json.Unmarshal(paramsData, &params); err != nil {
		logger.Fatal(err)
	}

	test, err := tester.New(ctx, params, config, logger)
	if err != nil {
		logger.Fatal(err)
	}
	if err := test.Execute(ctx); err != nil {
		logger.Fatal(err)
	}

	// write sheet to file
	var data bytes.Buffer
	writer := bufio.NewWriter(&data)
	if err := test.Sheet.File.Write(writer); err != nil {
		logger.Fatal(err)
	}
	writeResult(testName, data.Bytes())
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
