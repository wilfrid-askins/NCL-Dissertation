package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
	"wilfrid.me/archive-central/pkg/central"
	"wilfrid.me/archive-test/internal/tester"
)

// root is where the program will look for test configuration files.
const root = "./tests"

// tests is the list of all
var tests = []string{
	"test1",
	"test2",
	"test3",
	"external1",
        "external2",
}

func main() {
	ctx, cfn := context.WithTimeout(context.Background(), time.Minute*10)
	defer cfn()
	prodLog, _ := zap.NewProduction()
	logger := prodLog.Sugar()

	testName := pickTest()
	box := packr.New(testName, root+"/"+testName)

	// load config and create resolver
	config := central.Config{}
	configData, err := box.Find("central.json")
	if err != nil {
		logger.Fatal(err.Error())
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		logger.Fatal(err.Error())
	}

	// load test files
	params := tester.Config{}
	paramsData, err := box.Find("test.json")
	if err := json.Unmarshal(paramsData, &params); err != nil {
		logger.Fatal(err)
	}

	// create tester instance
	test, err := tester.New(ctx, params, config, logger)
	if err != nil {
		logger.Fatal(err)
	}
	// execute test
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

// writeResult writes bytes to a specified file.
func writeResult(testName string, data []byte) {
	fileName := fmt.Sprintf("%s-%s.xlsx", testName, time.Now().Format("2006-01-02T15-04-05"))
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

// pickTest will ask a user to pick a valid test name.
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
