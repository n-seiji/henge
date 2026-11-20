# ebi-x

ebi-x は、Slack で頼まれた作業を Codex に実行させ、結果を同じスレッドへ返す常駐ボットです。

```text
@tanaka  @ebi-x CI が落ちている原因を調べて
  └ ebi-x がコマンドを実行しています（go test）… 1/3 完了
  └ @ebi-x  原因はタイムゾーンの扱いでした。修正して PR を作成しました。…
            👀 次の PR を見守ります。CI の失敗やレビューがあれば、このスレッドで対応します。
```

## できること

- **依頼**：mention、ebi-x との DM、メッセージの「…」メニューから頼めます。スレッドに返信すれば、同じ Codex セッションで続きを頼めます。
- **作業の見える化**：作業中は Codex がいま何をしているかを表示し、「停止」で止められます。Home タブに自分の依頼の状況が並びます。
- **会話が止まった後の作業**：後で確認するフォローアップ、曜日と時刻を決めた定期実行、指定の時刻に1回知らせるリマインダー、作った PR の CI やレビューへの対応ができます。
- **成果物**：PDF・画像・pptx・CSV をスレッドに添付します。
- **安全な実行**：使える人とチャンネルを許可制にし、Codex はスレッドごとの作業場所とサンドボックスの中で動きます。外部への送信など確認が必要な操作の前には、依頼者に聞きます。

## はじめかた

必要なもの：Codex CLI 0.149.0 以上（`codex login` 済み）、Slack App を作れるワークスペース、常時動かしておける macOS か Linux。

1. インストールします。

   ```sh
   curl -fsSL https://raw.githubusercontent.com/n-seiji/henge/main/install.sh | sh
   ```

2. エージェント用のディレクトリを作り、`ebi-x init` を実行します。`.env` と `data/` はこのディレクトリから読みます。

   ```sh
   mkdir ~/ebi-x && cd ~/ebi-x && ebi-x init
   ```

   書き出された `slack-app-manifest.json` を Slack App の作成画面の「From a manifest」に貼り付けてワークスペースにインストールし、続く質問に Bot Token（`xoxb-`）、Socket Mode 用の App Token（`xapp-`）、自分の member ID、bot と話すチャンネルの ID を答えると、`.env` ができます。そのチャンネルが承認チャンネルになり、ほかの人やチャンネルはそこで Slack から許可します。

3. Codex にログインして起動します。

   ```sh
   codex login
   TZ=Asia/Tokyo ebi-x
   ```

4. 答えたチャンネルに ebi-x を招待し、`@ebi-x 生きてる？` と mention して応答を確かめます。

常駐させる方法と、ソースからのビルドは [docs/setup.md](docs/setup.md#0-インストール) にあります。

## ドキュメント

| ファイル | 内容 |
| --- | --- |
| [docs/setup.md](docs/setup.md) | 導入と設定（Slack App、`.env`、許可、運用） |
| [docs/usage.md](docs/usage.md) | Slack での使い方と、各機能の動き |
| [docs/design.md](docs/design.md) | 設計の決めごとと理由 |
| [examples/](examples/) | Slack App の manifest、常駐の雛形、操作ルールと playbook の例 |

## 開発

```sh
mise run test   # go test -race ./...
mise run lint   # go vet ./... と gofmt の確認
```

### 公開してはいけないものを止める

このリポジトリは公開されています。トークンなどの秘密と、社名やチャンネル名のような載せてはいけない言葉を、コミット前と CI で止めます。

```sh
mise run hooks      # コミットのたびに、ステージした変更とコミットメッセージを確かめる
mise run scan       # 全コミットの変更とメッセージを確かめる
mise run vulncheck  # 依存とコードに既知の脆弱性がないか確かめる
```

載せてはいけない言葉は、1行に1つずつリポジトリ直下の `.denylist` に書きます（Git には入りません。`#` で始まる行は無視、大文字小文字は区別しません）。CI では、同じ内容を Actions の secret `DENYLIST` に入れておくと、全コミットと PR のタイトル・本文を確かめます。

動いている ebi-x にコードの変更を反映するときは `mise run reload` を使います。実行中の作業を止めずに入れ替えます（[docs/setup.md](docs/setup.md#9-運用)）。
