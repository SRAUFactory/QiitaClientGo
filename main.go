package main

import (
	"encoding/json"
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
	CreatedAt string `json:"created_at"`
	Id        string
	Private   bool
	Tags      []Tag
	Title     string
	UpdatedAt string `json:"updated_at"`
	Url       string
	User      User
}

func errorHandler(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	url := "https://qiita.com/api/v2/authenticated_user/items?page=1&per_page=100"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("QIITA_API_TOKEN"))

	client := new(http.Client)
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)

	var items []Item
	err := json.Unmarshal(byteArray, &items)
	errorHandler(err)

	output, err := json.Marshal(items)
	errorHandler(err)
	fmt.Println(string(output))
}
