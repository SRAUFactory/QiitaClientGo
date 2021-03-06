package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Tag struct {
	Name string
}

type User struct {
	Description     string
	Location        string
	Name            string
	Organization    string
	ProfileImageUrl string `json:"profile_image_url"`
	WebsiteUrl      string `json:"website_url"`
}

type Item struct {
	CreatedAt     string `json:"created_at"`
	CommentsCount int    `json:"comments_count"`
	Id            string
	LikesCount    int `json:"likes_count"`
	Private       bool
	Tags          []Tag
	Title         string
	UpdatedAt     string `json:"updated_at"`
	Url           string
	User          User
}

var (
	outputType = flag.String("t", "stdout", "Invalid value are 'stdout', 'file' only.")
	outputFile = flag.String("f", "./qiita.json", "Set output file path.")
)

func errorHandler(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func outputResult(result []byte) {
	if *outputType == "file" {
		file, _ := os.Create(*outputFile)
		defer file.Close()
		file.Write(result)
		return
	}
	fmt.Println(string(result))
}

func getQiitaData() []byte {
	url := "https://qiita.com/api/v2/authenticated_user/items?page=1&per_page=100"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("QIITA_API_TOKEN"))

	client := new(http.Client)
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	byteArray, _ := ioutil.ReadAll(resp.Body)

	return byteArray
}

func convertJSONData(byteArray []byte) []byte {
	var items []Item
	err := json.Unmarshal(byteArray, &items)
	errorHandler(err)

	result, err := json.Marshal(items)
	errorHandler(err)

	return result
}

func main() {
	flag.Parse()
	byteArray := getQiitaData()
	output := convertJSONData(byteArray)
	outputResult(output)
}
