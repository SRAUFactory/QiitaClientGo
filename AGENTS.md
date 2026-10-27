# AGENTS.md

このファイルは、本リポジトリで作業するAIエージェント向けのガイドです。

## プロジェクト概要

QiitaClientGo (`qiita-client`) は、Qiita APIとローカルのMarkdownファイルを双方向に同期するGo製CLIツールです。

- **外部依存ゼロ**: Go標準ライブラリのみで実装している。サードパーティ製ライブラリは追加しない(サプライチェーンリスクを避けるため)。
- ライセンス: MIT

## ディレクトリ構成

ルート直下にパッケージ `main` のファイルを置くフラットな構成です。

| ファイル | 役割 |
|---|---|
| `main.go` | エントリポイント、サブコマンドの振り分け |
| `client.go` | Qiita APIクライアント(HTTP通信) |
| `frontmatter.go` | YAML Frontmatterの軽量パーサー/生成器 |
| `subcommand_fetch.go` | `fetch`: Qiita記事を取得してMarkdownとして保存 |
| `subcommand_push.go` | `push`: Markdownを新規投稿(POST)または更新(PATCH) |
| `subcommand_diff.go` | `diff`: `git diff --name-only` から変更されたMarkdownを検出 |
| `*_test.go` | 各ファイルに対応するテスト |
| `.github/workflows/release.yml` | タグpush時のリリースビルドとGitHub Release作成 |

## コマンド

```bash
# テスト
go test -v ./...

# 静的チェック・フォーマット
go vet ./...
gofmt -l .

# ビルド
go build -ldflags "-s -w" -o bin/qiita-client .
```

## コーディング規約

- 外部ライブラリを追加しない。`go.mod` に `require` が増える変更は行わない。
- 新しい機能や修正には、対応する `*_test.go` を追加または更新する。
- コミット前に `gofmt`、`go vet`、`go test ./...` を通す。
- 全HTTP通信にタイムアウト(30秒)を設定する方針を維持する。
- エラーは握りつぶさず、呼び出し元へ返すか、標準エラー出力へ分かりやすく出す。

## セキュリティ上の注意

- **認証トークン**: Qiitaの個人用アクセストークンは環境変数 `QIITA_TOKEN`(または `-token` オプション)で渡す。ソースコード、テスト、ログ、コミットに実際のトークンを含めない。
- **パストラバーサル対策**: `fetch` は記事タイトルをサニタイズし、基準ディレクトリに対する相対パスを検証してから書き込む。この処理を弱める変更をしない。
- **`-dry-run`**: `push` のドライランでは、実際のAPI呼び出しやファイル変更を行わない。この挙動を保つ。
- テストで実際のQiita APIを叩かない。HTTP通信はテスト用サーバーなどで置き換える。

## Frontmatter仕様

Markdownの先頭に、次のフィールドを持つYAML Frontmatterを置きます。すべて任意です。

| フィールド | 型 | 説明 |
|---|---|---|
| `title` | 文字列 | 記事タイトル(未指定時はファイル名) |
| `tags` | 配列 | 記事に付与するタグ |
| `qiita_id` | 文字列 | Qiita記事ID。空の場合は新規投稿の対象 |
| `private` | 真偽値 | 限定共有記事フラグ(既定値は `false`) |
| `updated_at` | 文字列 | Qiita側の最終更新日時(`fetch` 時に記録) |

仕様を変える場合は、`frontmatter.go`、そのテスト、READMEの仕様表をあわせて更新してください。

## リリース手順

リリースは、`v*` 形式のタグをpushすると `.github/workflows/release.yml` が起動して行われます。

- タグの作成・付け直しは、Skill `git-release-tag`(`.claude/skills/git-release-tag/SKILL.md`)の手順に従う。
- タグのpushでリリースが走るため、**ユーザーの明示的な承認なしにタグをpushしない**。
- ワークフローは、タグが指すコミットの内容で動く。ワークフローを修正した場合は、修正を含むコミットにタグを付け直す。
- ビルド対象は linux/amd64、linux/arm64、darwin/amd64、darwin/arm64。アーカイブ名は `qiita-client_<tag>_<os>_<arch>.tar.gz`。
- `checksums.txt` も同じリリースに添付される。READMEのCI例はこのチェックサムでSHA256検証を行っているため、アーカイブ名と形式を変えない。
- ビルド時に `-X main.Version=<tag>` でバージョンを埋め込む。`main.go` に対応する変数が必要。

## 作業時の方針

- 変更は最小限にとどめ、依頼された範囲を超えるリファクタリングをしない。
- README、`AGENTS.md`、ワークフローの記述が実装とずれないよう、挙動を変えたら同じ変更の中でドキュメントも更新する。
- 破壊的な操作(タグの削除、`push --force`、記事のQiitaへの投稿・更新)は、必ず事前に確認を取る。
