# bedrock-usage 要件定義書

## 1. プロジェクト概要

### 1.1 目的
Claude Code + Amazon Bedrock API を利用する際の **トークン使用量** と **予想利用料金** を、IAM ユーザー/ロール単位で確認できる Go 製 CLI ツールを開発する。

### 1.2 背景
Claude Code を Bedrock API 経由で使用する場合、日々のトークン消費量や料金を素早く把握する手段が限られている。特に複数ユーザーが同一 AWS アカウントを共有する場合、誰がどれだけ使ったかを把握する必要がある。本ツールにより、Bedrock の Model Invocation Logging (CloudWatch Logs) からリクエスト単位のログを取得・集計し、ターミナル上で使用状況をすぐに確認できるようにする。

### 1.3 ツール名
`bedrock-usage`

---

## 2. 前提条件

### 2.1 Model Invocation Logging の有効化
本ツールを使用するには、**Bedrock を利用するリージョンごとに** Model Invocation Logging を事前に有効化しておく必要がある。

- Bedrock コンソール → Settings → Model invocation logging を有効化
- ログ送信先: **CloudWatch Logs** (必須)
- データタイプ: **Text** を選択 (最低限必要。イメージ・埋め込み・ビデオは任意)
- ロググループ名: `bedrock/modelinvocations` (推奨。`aws/` 始まりは AWS 予約済みのため使用不可)
- サービスロール: Bedrock が CloudWatch Logs に書き込むための IAM ロールが必要

> **注意**: ログ記録はリージョン固有の設定である。Claude Code が Bedrock を呼び出すリージョン (例: `us-east-1`) でログ記録を有効化すること。

### 2.2 ログスキーマ
Bedrock が CloudWatch Logs に出力するログの構造 (本ツールが依存するフィールド):

```json
{
  "schemaType": "ModelInvocationLog",
  "timestamp": "2026-03-17T10:10:49Z",
  "accountId": "123456789012",
  "identity": {
    "arn": "arn:aws:sts::123456789012:assumed-role/RoleName/user@example.com"
  },
  "region": "us-east-1",
  "requestId": "c91de5d3-...",
  "operation": "Converse",
  "modelId": "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-opus-4-6-v1",
  "input": {
    "inputTokenCount": 5757
  },
  "output": {
    "outputTokenCount": 639
  }
}
```

> **注意**: `modelId` は inference profile ARN 形式 (`arn:aws:bedrock:...:inference-profile/us.anthropic.claude-...`) で記録される。本ツールでは表示時に `us.` / `global.` 等のリージョンプレフィックスを除去し、短縮表示する。

**使用するフィールド**:
| フィールド | 用途 |
|---|---|
| `identity.arn` | ユーザー/ロール別フィルタ・集計 |
| `modelId` | モデル別集計 (inference profile ARN 形式) |
| `input.inputTokenCount` | 入力トークン数 |
| `output.outputTokenCount` | 出力トークン数 |
| `timestamp` | 期間フィルタ・日次集計 |

---

## 3. 機能要件

### 3.1 トークン使用量の取得
- **データソース**: CloudWatch Logs (Model Invocation Logging)
- **取得方法**: CloudWatch Logs Insights API (`StartQuery` / `GetQueryResults`)
- **集計内容**:
  - 入力トークン数の合計
  - 出力トークン数の合計
  - リクエスト回数 (Invocations)
- **集計軸**:
  - モデル別 (`modelId`)
  - ユーザー別 (`identity.arn`)
  - 日別 (`timestamp`)

### 3.2 予想利用料金の計算
- 取得したトークン数に、モデルごとの単価を掛けて料金を算出する
- デフォルトの単価をバイナリに埋め込む
- inference profile ARN からモデル名を正規化し、単価テーブルと照合する
- 対応モデル (Bedrock On-Demand / Standard Tier):
  | モデル | マッチキー | 入力 (per 1K tokens) | 出力 (per 1K tokens) |
  |---|---|---|---|
  | Claude Opus 4 / 4.6 | `anthropic.claude-opus-4` | $0.015 | $0.075 |
  | Claude Sonnet 4 | `anthropic.claude-sonnet-4` | $0.003 | $0.015 |
  | Claude Sonnet 3.5 | `anthropic.claude-sonnet-3-5` | $0.003 | $0.015 |
  | Claude Haiku 4.5 | `anthropic.claude-haiku-4-5` | $0.0008 | $0.004 |
  | Claude Haiku 3.5 | `anthropic.claude-haiku-3-5` | $0.0008 | $0.004 |
  | Claude 3 Opus | `anthropic.claude-3-opus` | $0.015 | $0.075 |
  | Claude 3 Sonnet | `anthropic.claude-3-sonnet` | $0.003 | $0.015 |
  | Claude 3 Haiku | `anthropic.claude-3-haiku` | $0.00025 | $0.00125 |

  ※ 実際の価格は AWS の最新料金ページを参照のこと
- **制限事項**: `--by-user` 表示時はモデル情報が集計に含まれないため、料金計算は行わない ($0.00 と表示)

### 3.3 表示情報
以下の情報をテーブル形式で表示する:

#### サマリー表示 (デフォルト)
```
Bedrock Usage Summary (2026-03-17)

┌──────────────────────────────────────────┬──────────────┬───────────────┬─────────────┬───────────┐
│ MODEL                                    │ INPUT TOKENS │ OUTPUT TOKENS │ INVOCATIONS │ EST. COST │
├──────────────────────────────────────────┼──────────────┼───────────────┼─────────────┼───────────┤
│ anthropic.claude-haiku-4-5-20251001-v1:0 │          652 │            35 │           2 │     $0.00 │
│             anthropic.claude-opus-4-6-v1 │          301 │         1,008 │           3 │     $0.08 │
├──────────────────────────────────────────┼──────────────┼───────────────┼─────────────┼───────────┤
│                                    Total │          953 │         1,043 │           5 │     $0.08 │
└──────────────────────────────────────────┴──────────────┴───────────────┴─────────────┴───────────┘
```

#### ユーザー別表示 (`--by-user` フラグ)
```
Bedrock Usage by User (2026-03-17)

┌──────────────────────────────────────────────────┬──────────────┬───────────────┬─────────────┬───────────┐
│ USER                                             │ INPUT TOKENS │ OUTPUT TOKENS │ INVOCATIONS │ EST. COST │
├──────────────────────────────────────────────────┼──────────────┼───────────────┼─────────────┼───────────┤
│ AWSReservedSSO_.../forza0720+aws@gmail.com        │          953 │         1,043 │           5 │     $0.00 │
├──────────────────────────────────────────────────┼──────────────┼───────────────┼─────────────┼───────────┤
│                                            Total │          953 │         1,043 │           5 │     $0.00 │
└──────────────────────────────────────────────────┴──────────────┴───────────────┴─────────────┴───────────┘
```
> ※ ユーザー別表示ではモデル別の内訳がないため、Est. Cost は $0.00 となる

#### 日次推移表示 (`--daily` フラグ)
```
Daily Breakdown (2026-03-15 ~ 2026-03-17)

┌────────────┬──────────────┬───────────────┬─────────────┬───────────┐
│ DATE       │ INPUT TOKENS │ OUTPUT TOKENS │ INVOCATIONS │ EST. COST │
├────────────┼──────────────┼───────────────┼─────────────┼───────────┤
│ 2026-03-15 │       45,200 │        18,300 │          30 │     $0.41 │
│ 2026-03-16 │       38,100 │        14,500 │          25 │     $0.33 │
│ 2026-03-17 │       42,100 │        15,400 │          30 │     $0.36 │
├────────────┼──────────────┼───────────────┼─────────────┼───────────┤
│      Total │      125,400 │        48,200 │          85 │     $1.10 │
└────────────┴──────────────┴───────────────┴─────────────┴───────────┘
```

### 3.4 期間指定
プリセットから選択する方式:

| フラグ | 説明 |
|---|---|
| `--period today` (デフォルト) | 本日の使用量 |
| `--period week` | 今週 (月曜〜今日) の使用量 |
| `--period month` | 今月 (1日〜今日) の使用量 |

### 3.5 AWS 設定
- AWS SDK for Go v2 のデフォルト認証チェーンを使用する
  - 環境変数 (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `AWS_PROFILE`)
  - 共有認証情報ファイル (`~/.aws/credentials`)
  - 共有設定ファイル (`~/.aws/config`)
  - AWS SSO / IAM Identity Center
  - IAM ロール (EC2/ECS 上での実行時)
- リージョンは SDK のデフォルト解決順に従う

### 3.6 ロググループ指定
- デフォルトのロググループ名: `bedrock/modelinvocations`
- `--log-group` フラグでカスタムロググループ名を指定可能

> **注意**: `aws/` で始まるロググループ名は AWS に予約されているため使用不可。

---

## 4. 非機能要件

### 4.1 プラットフォーム
- マルチプラットフォーム対応のシングルバイナリ
- 対応 OS/Arch:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/amd64`
  - `darwin/arm64`
  - `windows/amd64`

### 4.2 ビルド・配布
- Go Modules でのビルド
- `go install` でインストール可能
- GitHub Releases での配布 (GoReleaser 利用)

### 4.3 依存ライブラリ
- `aws-sdk-go-v2` — AWS API 呼び出し (CloudWatch Logs Insights)
- `cobra` — CLI フレームワーク
- `olekukonko/tablewriter` v1 — テーブル表示

### 4.4 必要な IAM 権限 (ツール実行ユーザー向け)
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:StartQuery",
        "logs:GetQueryResults",
        "logs:DescribeLogGroups"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:bedrock/modelinvocations:*"
    }
  ]
}
```

### 4.5 必要な IAM ロール (Bedrock ログ出力用)
Model Invocation Logging で Bedrock が CloudWatch Logs に書き込むためのサービスロール:

**許可ポリシー**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:<ACCOUNT_ID>:log-group:bedrock/modelinvocations:*"
    }
  ]
}
```

**信頼ポリシー**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "bedrock.amazonaws.com" },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": { "aws:SourceAccount": "<ACCOUNT_ID>" },
        "ArnLike": { "aws:SourceArn": "arn:aws:bedrock:*:<ACCOUNT_ID>:*" }
      }
    }
  ]
}
```

---

## 5. CLI インターフェース

### 5.1 基本コマンド

```bash
# 今日のサマリーを表示 (全ユーザー合計、デフォルト)
AWS_PROFILE=learn AWS_REGION=us-east-1 bedrock-usage

# 期間を指定
bedrock-usage --period week
bedrock-usage --period month

# 特定ユーザーのみ表示 (部分一致)
bedrock-usage --user alice
bedrock-usage --user forza0720

# ユーザー別内訳を表示
bedrock-usage --by-user

# 日次推移を表示
bedrock-usage --daily

# 特定モデルのみ表示 (部分一致)
bedrock-usage --model claude-opus
bedrock-usage --model claude-haiku

# 組み合わせ
bedrock-usage --period month --daily --user alice --model claude-opus

# カスタムロググループ名
bedrock-usage --log-group my-bedrock-logs
```

### 5.2 グローバルフラグ

| フラグ | 短縮 | 説明 |
|---|---|---|
| `--period` | `-p` | 期間指定 (`today`, `week`, `month`) |
| `--daily` | `-d` | 日次推移を表示 |
| `--model` | `-m` | モデルID でフィルタ (部分一致) |
| `--user` | `-u` | IAM ユーザー/ロール ARN でフィルタ (部分一致) |
| `--by-user` | | ユーザー別の内訳表示 |
| `--log-group` | `-l` | CloudWatch Logs のロググループ名 |
| `--help` | `-h` | ヘルプ表示 |
| `--version` | `-v` | バージョン表示 |

---

## 6. 内部設計

### 6.1 モデル名の正規化
inference profile ARN からモデル名を抽出する処理:

```
入力: arn:aws:bedrock:us-east-1:123:inference-profile/us.anthropic.claude-opus-4-6-v1
  ↓ 最後の "/" 以降を取得
  us.anthropic.claude-opus-4-6-v1
  ↓ リージョンプレフィックス (us. / global. / eu. / ap.) を除去
  anthropic.claude-opus-4-6-v1
```

### 6.2 CloudWatch Logs Insights クエリ

#### サマリー取得
```
fields @timestamp, modelId, input.inputTokenCount as inputTokens, output.outputTokenCount as outputTokens, identity.arn
| stats sum(inputTokens) as totalInput, sum(outputTokens) as totalOutput, count(*) as invocations by modelId
| sort modelId
```

#### ユーザー別取得
```
fields @timestamp, modelId, input.inputTokenCount as inputTokens, output.outputTokenCount as outputTokens, identity.arn
| stats sum(inputTokens) as totalInput, sum(outputTokens) as totalOutput, count(*) as invocations by identity.arn
| sort identity.arn
```

#### 日次推移取得
```
fields @timestamp, modelId, input.inputTokenCount as inputTokens, output.outputTokenCount as outputTokens, identity.arn
| stats sum(inputTokens) as totalInput, sum(outputTokens) as totalOutput, count(*) as invocations by datefloor(@timestamp, 1d) as date
| sort date asc
```

---

## 7. プロジェクト構成

```
bedrock-usage/
├── main.go                      # エントリーポイント
├── cmd/
│   └── root.go                  # CLI 定義 (cobra) とメイン実行ロジック
├── internal/
│   ├── query/
│   │   ├── client.go            # CloudWatch Logs Insights クエリ実行・結果パース
│   │   └── builder.go           # クエリ文字列の組み立て
│   ├── pricing/
│   │   └── pricing.go           # デフォルト単価定義・モデル名正規化・料金計算
│   └── display/
│       └── table.go             # テーブル表示・ARN 短縮表示
├── go.mod
├── go.sum
├── .goreleaser.yaml             # マルチプラットフォームビルド設定
└── docs/
    └── requirements.md          # 本ドキュメント
```

---

## 8. 将来的な拡張候補 (スコープ外)

以下は v1 のスコープには含めないが、将来的に検討する:

- `--by-user` 表示時のモデル別クロス集計による料金計算対応
- `--json` フラグによる JSON 出力
- `--from` / `--to` による任意日付範囲指定
- `--profile` / `--region` フラグによる AWS 設定上書き
- 設定ファイルによるカスタム単価の上書き (`~/.bedrock-usage/pricing.yaml`)
- CSV エクスポート
- 予算アラート / 閾値超過の警告表示
- 複数リージョンの集約表示
