# QiitaClientGo (`qiita-client`)

Qiita APIとローカルMarkdownファイルを双方向同期するための高速・安全なGo製CLIツールです。

サードパーティ製ライブラリを一切含まず、**Go標準ライブラリのみ（外部依存ゼロ）**で実装されています。サプライチェーン攻撃リスクを排除し、GitHub Actions等のCI/CDパイプラインやローカル環境で安全に動作します。

---

## 主な特徴

- 🔒 **外部依存ゼロ**: 標準ライブラリのみで完結（サプライチェーンリスクの完全排除）
- 📄 **Frontmatter解析**: YAML Frontmatter（`title`, `tags`, `qiita_id`, `private`, `updated_at`）の軽量パーサー/生成器を内蔵
- 🔄 **双方向同期**:
  - `fetch`: Qiita APIから記事を全件取得し、Frontmatter付きMarkdownとして保存（更新日時比較で差分更新、パストラバーサル防止）
  - `push`: MarkdownをQiitaへ新規投稿・更新（新規投稿時は生成された `qiita_id` をファイルへ自動書き戻し）
  - `diff`: Gitの差分（`git diff`）から変更された対象Markdownファイルを検出
- 🚀 **UNIXパイプライン親和性**: `diff` から `push` へのパイプ連携（`qiita-client diff | qiita-client push`）をサポート
- ⏱️ **通信堅牢化**: 全HTTP通信にタイムアウト（30秒）を設定

---

## インストール

### リリースバイナリのダウンロード
[GitHub Releases](https://github.com/SRAUFactory/QiitaClientGo/releases) より、お使いの環境（Linux/macOS, amd64/arm64）に合わせたアーカイブと `checksums.txt` をダウンロードし、SHA256検証を行って展開してください。

```bash
# 例: Linux amd64
tar -xzf qiita-client_v1.0.0_linux_amd64.tar.gz
chmod +x qiita-client
sudo mv qiita-client /usr/local/bin/
```

### ソースからビルド
```bash
git clone https://github.com/SRAUFactory/QiitaClientGo.git
cd QiitaClientGo
go build -ldflags "-s -w" -o bin/qiita-client .
```

---

## 認証

Qiitaの個人用アクセストークン（`read_qiita`, `write_qiita` スコープ）を環境変数 `QIITA_TOKEN` に設定します。

```bash
export QIITA_TOKEN="your_qiita_access_token_here"
```

※ `-token` オプションでコマンドラインから直接渡すことも可能です。

---

## 使い方

```bash
qiita-client <command> [options] [arguments]
```

### 1. `fetch`: Qiita記事の取得・Markdown保存

Qiita APIから全記事を取得し、指定ディレクトリ配下にMarkdownファイルとして保存します。

```bash
# デフォルト（./qiita ディレクトリに出力）
qiita-client fetch

# 出力先ディレクトリを指定
qiita-client fetch -dir path/to/articles
```

- **更新判定**: ローカルファイルの `updated_at` とQiita側の更新日時を比較し、変更のない記事はスキップします。
- **初回紐付け**: `qiita_id` が未設定でも、ファイル名（拡張子除く）と記事タイトルが一致する場合は自動的に紐付けを行い `qiita_id` を書き込みます。
- **セキュリティ**: 記事タイトル文字列のサニタイズおよび基準ディレクトリに対する相対パス検証（パストラバーサル防止）を徹底しています。

### 2. `push`: Markdown記事のQiita投稿・更新

指定されたMarkdownファイルを読み込み、新規投稿または更新を行います。

```bash
# ファイルを引数で直接指定
qiita-client push qiita/article1.md qiita/article2.md

# 標準入力からパイプで渡す
echo "qiita/article1.md" | qiita-client push

# diffと組み合わせて変更記事のみをpush
qiita-client diff -dir qiita | qiita-client push

# ドライラン（実際のAPI呼び出しやファイル変更を行わずに確認）
qiita-client push -dry-run qiita/article1.md
```

- **新規投稿**: Frontmatter内に `qiita_id` が存在しない場合、Qiitaへ新規投稿（POST）し、発行された記事IDをファイルのFrontmatterに書き戻します。
- **更新**: `qiita_id` が存在する場合、該当記事を更新（PATCH）します。

### 3. `diff`: 変更されたMarkdownファイルの検出

Git差分（`git diff --name-only`）を利用して、対象ディレクトリ内で変更・追加されたMarkdownファイル一覧を出力します。

```bash
# 直前コミット（HEAD~1）との差分を検出
qiita-client diff -dir qiita

# 比較ベースを指定（例: origin/main）
qiita-client diff -dir qiita -base origin/main
```

---

## Markdown Frontmatter仕様

本ツールで扱うMarkdownファイルは、ファイルの先頭に以下のYAML Frontmatterを含みます。

```markdown
---
title: "記事のタイトル"
tags:
  - Go
  - CLI
qiita_id: "e59fda709e5ed65eb489"
private: false
updated_at: '2026-10-27T12:00:00+09:00'
---

ここから記事の本文が始まります。
```

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `title` | 文字列 | 任意 | 記事タイトル（未指定時はファイル名を使用） |
| `tags` | 配列 | 任意 | 記事に付与するタグ一覧 |
| `qiita_id` | 文字列 | 任意 | Qiita記事ID。空の場合は新規投稿対象となります |
| `private` | 真偽値 | 任意 | 限定共有記事フラグ（デフォルト: `false`） |
| `updated_at` | 文字列 | 任意 | Qiita側での最終更新日時（fetch時に記録） |

---

## CI/CD ワークフロー例 (GitHub Actions)

### 記事プッシュ時同期 (`sync_to_qiita.yml`)

```yaml
name: Sync to Qiita

on:
  push:
    branches:
      - main
    paths:
      - 'qiita/*.md'

jobs:
  sync:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          fetch-depth: 2

      - name: Download qiita-client
        run: |
          VERSION="v1.0.0"
          curl -sSL -o qiita-client.tar.gz "https://github.com/SRAUFactory/QiitaClientGo/releases/download/${VERSION}/qiita-client_${VERSION}_linux_amd64.tar.gz"
          curl -sSL -o checksums.txt "https://github.com/SRAUFactory/QiitaClientGo/releases/download/${VERSION}/checksums.txt"
          grep "qiita-client_${VERSION}_linux_amd64.tar.gz" checksums.txt | sha256sum -c -
          tar -xzf qiita-client.tar.gz
          chmod +x qiita-client

      - name: Sync modified articles
        env:
          QIITA_TOKEN: ${{ secrets.QIITA_TOKEN }}
        run: |
          ./qiita-client diff -dir qiita | ./qiita-client push

      - name: Commit qiita_id writebacks if any
        run: |
          if [ -n "$(git status --porcelain qiita/)" ]; then
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            git add qiita/
            git commit -m "chore: write back qiita_id [skip ci]"
            git push
          fi
```

---

## ライセンス

MIT License
