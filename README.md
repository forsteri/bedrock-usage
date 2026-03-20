# bedrock-usage

Amazon Bedrock のトークン使用量と予想利用料金を確認する CLI ツール。

CloudWatch Logs (Model Invocation Logging) からリクエスト単位のログを取得・集計し、モデル別・ユーザー別・日次の使用状況をターミナル上で確認できます。

## 前提条件

### Model Invocation Logging の有効化

Bedrock を利用するリージョンで Model Invocation Logging を有効化しておく必要があります。

1. Bedrock コンソール → Settings → Model invocation logging を有効化
2. ログ送信先: **CloudWatch Logs**
3. ロググループ名: `bedrock/modelinvocations` (推奨)

### IAM 権限

ツール実行ユーザーに以下の権限が必要です:

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

## インストール

### go install

```bash
go install github.com/forsteri/bedrock-usage@latest
```

### GitHub Releases

[Releases ページ](https://github.com/forsteri/bedrock-usage/releases) からバイナリをダウンロードできます。

## 使い方

### 基本

```bash
# 今日のモデル別サマリーを表示
bedrock-usage

# AWS プロファイルを指定
bedrock-usage --profile my-profile

# リージョンを指定 (デフォルト: us-east-1)
bedrock-usage --region ap-northeast-1
```

### 期間指定

```bash
# 今日 (デフォルト)
bedrock-usage --period today

# 今週 (月曜〜今日)
bedrock-usage --period week

# 今月 (1日〜今日)
bedrock-usage --period month
```

### 表示モード

```bash
# モデル別サマリー (デフォルト)
bedrock-usage

# ユーザー別の内訳
bedrock-usage --by-user

# 日次推移
bedrock-usage --daily
```

### フィルタ

```bash
# 特定モデルのみ (部分一致)
bedrock-usage --model claude-opus

# 特定ユーザーのみ (部分一致)
bedrock-usage --user alice
```

### 組み合わせ例

```bash
# 今月の日次推移を Opus モデルに絞って表示
bedrock-usage --period month --daily --model claude-opus

# 今週の特定ユーザーの使用量
bedrock-usage --period week --user forza0720

# プロファイル指定で今月のユーザー別集計
bedrock-usage --profile learn --period month --by-user
```

## 出力例

### モデル別サマリー

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

### 日次推移

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

## フラグ一覧

| フラグ | 短縮 | デフォルト | 説明 |
|---|---|---|---|
| `--period` | `-p` | `today` | 期間指定 (`today`, `week`, `month`) |
| `--daily` | `-d` | `false` | 日次推移を表示 |
| `--model` | `-m` | | モデルID でフィルタ (部分一致) |
| `--user` | `-u` | | IAM ユーザー/ロール ARN でフィルタ (部分一致) |
| `--by-user` | | `false` | ユーザー別の内訳を表示 |
| `--log-group` | `-l` | `bedrock/modelinvocations` | CloudWatch Logs のロググループ名 |
| `--profile` | | | AWS プロファイル名 |
| `--region` | | `us-east-1` | AWS リージョン |
| `--prices` | | `false` | 内蔵の単価テーブルを表示 |

## 料金計算について

- モデルごとの単価がバイナリに埋め込まれており、トークン数から予想料金を算出します
- `--prices` フラグで現在の内蔵単価テーブルを確認できます
- `--by-user` 表示時はモデル情報を含まないため、料金は `$0.00` と表示されます
- 料金はあくまで概算です。正確な請求額は AWS Billing を確認してください

## ライセンス

MIT
