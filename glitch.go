package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func getConfig() map[string]map[string]int {
	config := map[string]map[string]int{
		"avi": map[string]int{
			"val":    0,
			"repeat": 100,
			"off":    10000,
			"freq":   1,
			"start":  0,
			"end":    100,
		},
		"mkv": map[string]int{
			"val":    0,
			"repeat": 100,
			"off":    10000,
			"freq":   1,
			"start":  0,
			"end":    100,
		},
		"mp4": map[string]int{
			"val":    0,
			"repeat": 100,
			"freq":   1,
			"start":  0,
			"end":    100,
			"left":   0,
			"right":  100,
		},
		"mov": map[string]int{
			"val":    0,
			"repeat": 100,
			"freq":   1,
			"start":  0,
			"end":    100,
			"left":   0,
			"right":  100,
		},
	}

	return config
}

func exitGracefully(errorStr string) {
	fmt.Println(errorStr)
	os.Exit(1)
}

func checkIfValidFile(filename string) (bool, string) {
	// Make this cleaner
	if fileExtension := filepath.Ext(filename); fileExtension != ".avi" && fileExtension != ".mkv" && fileExtension != ".mp4" && fileExtension != ".mov" {
		return false, fmt.Sprintf("file extention %s cannot be glitched", fileExtension)
	}

	if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
		return false, fmt.Sprintf("file %s does not exist", filename)
	}

	return true, ""
}

func checkArgs() []string {
	flag.Parse()
	parsedArguments := flag.Args()

	if len(parsedArguments) < 1 {
		flag.PrintDefaults()
		exitGracefully("You must provide at least one video file to glitch.")
	}

	for _, str := range parsedArguments {
		if valid, errorString := checkIfValidFile(str); !valid {
			exitGracefully(errorString)
		}
	}

	return parsedArguments
}

func getMpegDataSect(data string) int {
	i := 0

	for ; i < len(data); i++ {
		if i+3 > len(data) {
			continue
		}
		if data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 {
			return i
		}
	}

	return i
}

func glitchMov(rawData string, isMov bool) string {
	config := getConfig()
	var val, repeat, freq, start, end, left, right int

	if isMov {
		val, repeat, freq = config["mov"]["val"], config["mov"]["repeat"], config["mov"]["freq"]
		start, end = config["mov"]["start"], config["mov"]["end"]
		left, right = config["mov"]["left"], config["mov"]["right"]
	} else {
		val, repeat, freq = config["mp4"]["val"], config["mp4"]["repeat"], config["mp4"]["freq"]
		start, end = config["mp4"]["start"], config["mp4"]["end"]
		left, right = config["mp4"]["left"], config["mp4"]["right"]
	}

	if start >= end {
		exitGracefully("Start value must be less than end value. See config file")
	}

	x := 0
	mpegStart := strings.Index(rawData, "mdat") + 4
	startDataLength := mpegStart + int(len(rawData)-mpegStart)*(start/100)
	endDataLength := int(len(rawData)-mpegStart) * (end / 100)

	for ; startDataLength < endDataLength; startDataLength++ {
		if rawData[startDataLength] == 0 && rawData[startDataLength+1] == 0 && rawData[startDataLength+2] == 1 && (rawData[startDataLength+3]&0x1f) != 5 && x%freq == 0 {
			splitData := rawData[startDataLength+3:]

			nextSect := getMpegDataSect(splitData)
			leftNextSect := int(nextSect * (left / 100))
			rightNextSect := int(nextSect * (right / 100))

			for ; leftNextSect < rightNextSect; leftNextSect++ {
				if leftNextSect%(repeat*100) == 0 {
					rawData = rawData[:startDataLength+leftNextSect] + strconv.Itoa(val) + rawData[(startDataLength+leftNextSect)+1:]
				}
			}
		}
		x++
	}

	fmt.Println(x + mpegStart)

	return rawData
}

func glitchAvi(rawData string) string {
	config := getConfig()
	val, repeat := config["avi"]["val"], config["avi"]["repeat"]
	off, freq := config["avi"]["off"], config["avi"]["freq"]
	start, end := config["avi"]["start"], config["avi"]["end"]

	x := 0
	aviStart := strings.Index(rawData, "movi") + 4
	startDataLength := aviStart + int(len(rawData)-aviStart)*(start/100)
	endDataLength := int(len(rawData)-aviStart) * (end / 100)

	for ; startDataLength < endDataLength; startDataLength++ {
		if x%freq == 0 {
			for j := 0; j < repeat; j++ {
				rawData = rawData[:startDataLength+off*(1+j)] + strconv.Itoa(val) + rawData[(startDataLength+off*(1+j))+1:]
			}
		}
		x++
	}

	return rawData
}

func glitchMkv(rawData string) string {
	config := getConfig()
	val, repeat := config["mkv"]["val"], config["mkv"]["repeat"]
	off, freq := config["mkv"]["off"], config["mkv"]["freq"]
	start, end := config["mkv"]["start"], config["mkv"]["end"]

	x := 0
	startDataLength := len(rawData) * (start / 100)
	endDataLength := len(rawData) * (end / 100)

	for ; startDataLength < endDataLength; startDataLength++ {
		if rawData[startDataLength] == 31 && rawData[startDataLength+1] == 67 && rawData[startDataLength+2] == 182 && rawData[startDataLength+3] == 117 && x%freq == 0 {
			for j := 0; j < repeat; j++ {
				rawData = rawData[:startDataLength+off*(1+j)] + strconv.Itoa(val) + rawData[(startDataLength+off*(1+j))+1:]
			}
		}
		x++
	}

	return rawData
}

func main() {
	args := checkArgs()

	for _, file := range args {
		fileExtension := filepath.Ext(file)
		fileName := strings.Replace(file, fileExtension, "", 1)
		fileReader, readErr := os.Open(file)
		fileWriter, writeErr := os.Create(fileName + "_go_glitched" + fileExtension)

		if readErr != nil || writeErr != nil {
			exitGracefully(fmt.Sprintf("error on reading %s", file))
		}

		defer fileReader.Close()
		defer fileWriter.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(fileReader)
		rawData := buf.String()

		var outputData string

		switch fileExtension {
		case ".mov":
			outputData = glitchMov(rawData, true)
		case ".mp4":
			outputData = glitchMov(rawData, false)
		case ".avi":
			outputData = glitchAvi(rawData)
		case ".mkv":
			outputData = glitchMkv(rawData)
		}

		_, outputErr := fileWriter.WriteString(outputData)

		if outputErr != nil {
			exitGracefully(fmt.Sprintf("error on output %s", file))
		}

		fmt.Println("done")
	}
}
