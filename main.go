package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Group struct {
	CreatedAt string `json:"created_at"`
	Id        int
	Name      string
	Private   bool
	UpdatedAt string `json:"updated_at"`
	UrlName   string `json:"url_name"`
}

type Tag struct {
	Name     string
	Versions []string
}

type User struct {
	Description       string
	Facebook_id       string `json:"facebook_id"`
	FolloweesCount    int    `json:"followees_count"`
	FollowersCount    int    `json:"followers_count"`
	GithubLoginName   string `json:"github_login_name"`
	Id                string
	itemsCount        int    `json:"items_count"`
	linkedinId        string `json:"linkedin_id"`
	Location          string
	Name              string
	Organization      string
	PermanentId       int    `json:"permanent_id"`
	ProfileImageUrl   string `json:"profile_image_url"`
	TwitterScreenName string `json:"twitter_screen_name"`
	WebsiteUrl        string `json:"website_url"`
}

type Item struct {
	RenderedBody   string `json:"rendered_body"`
	Body           string
	Coediting      bool
	CommentsCount  int    `json:"comments_count"`
	CreatedAt      string `json:"created_at"`
	Group          Group
	Id             string
	LikesCount     int `json:"likes_count"`
	Private        bool
	ReactionsCount int `json:"reactions_count"`
	Tags           []Tag
	Title          string
	UpdatedAt      string `json:"updated_at"`
	Url            string
	User           User
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
	fmt.Println(err)
	for key, value := range items {
		fmt.Println(key, value.Title, value.UpdatedAt)
	}

}
