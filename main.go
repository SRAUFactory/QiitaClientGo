package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io" // io/ioutil の代わりに io パッケージを使用
	"net/http"
	"os"
)

type Tag struct {
	Name string
}

type User struct {
	Description     string `json:"description"`
	Location        string `json:"location"`
	Name            string `json:"name"`
	Organization    string `json:"organization"`
	ProfileImageUrl string `json:"profile_image_url"`
	WebsiteUrl      string `json:"website_url"`
}

// structタグに合わせてフィールド名を修正（omitemptyなども考慮する場合があるが、今回は元の構造を維持）
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

// エラーハンドリング関数は、直接エラーをログ出力し、呼び出し元で処理を止めさせる形に修正または単純化することが望ましい。
// 今回は、致命的なエラーをログ出力する形に調整。
func errorHandler(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func outputResult(result []byte) {
	if *outputType == "file" {
		// エラーを無視せず、適切にチェックする
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Printf("Error creating file: %v\n", err)
			return
		}
		defer file.Close()

		// Writeの戻り値(N, err)もチェックする
		_, writeErr := file.Write(result)
		if writeErr != nil {
			fmt.Printf("Error writing to file: %v\n", writeErr)
		}
		return
	}
	fmt.Println(string(result))
}

// getQiitaData は、エラーを返すようにシグネチャを変更
func getQiitaData() ([]byte, error) {
	url := "https://qiita.com/api/v2/authenticated_user/items?page=1&per_page=100"

	// 1. http.NewRequestのエラーをチェック
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("http requestの作成に失敗しました: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("QIITA_API_TOKEN"))

	client := &http.Client{}

	// 2. client.Do(req) のエラーをチェック
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpリクエストの実行に失敗しました: %w", err)
	}
	// リソース解放処理は、必ずrespがnilでないことを保証してから行うべき
	defer resp.Body.Close()

	// 3. ioutil.ReadAll の代わりに io.ReadAll を使用し、エラーをチェック
	byteArray, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンスボディの読み取りに失敗しました: %w", err)
	}

	return byteArray, nil
}

// convertJSONData もエラーを返すようにシグネチャを変更
func convertJSONData(byteArray []byte) ([]byte, error) {
	var items []Item

	// JSONパース時のエラーチェック
	err := json.Unmarshal(byteArray, &items)
	if err != nil {
		return nil, fmt.Errorf("JSONデータ解析に失敗しました: %w", err)
	}

	// JSONマーシャリング時のエラーチェック
	result, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("JSONデータ変換に失敗しました: %w", err)
	}

	return result, nil
}

func main() {
	flag.Parse()

	// getQiitaData からエラーを捕捉する
	byteArray, err := getQiitaData()
	if err != nil {
		fmt.Printf("致命的なエラー: データ取得に失敗しました。\n%v\n", err)
		return
	}

	// convertJSONData からエラーを捕捉する
	output, err := convertJSONData(byteArray)
	if err != nil {
		fmt.Printf("致命的なエラー: データ処理に失敗しました。\n%v\n", err)
		return
	}

	outputResult(output)
}
