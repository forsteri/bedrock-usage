package pricing

import "strings"

type ModelPrice struct {
	InputPer1KTokens  float64
	OutputPer1KTokens float64
}

type PricingEntry struct {
	Prefix string
	Price  ModelPrice
}

// DefaultPrices はモデル別の単価定義。
// より具体的なプレフィックスを先に配置し、最長一致で正しい単価を返す。
var DefaultPrices = []PricingEntry{
	// Opus 4.6 / 4.5: $5 / $25 per 1M tokens
	{"anthropic.claude-opus-4-6", ModelPrice{InputPer1KTokens: 0.005, OutputPer1KTokens: 0.025}},
	{"anthropic.claude-opus-4-5", ModelPrice{InputPer1KTokens: 0.005, OutputPer1KTokens: 0.025}},
	// Opus 4 / 4.1: $15 / $75 per 1M tokens
	{"anthropic.claude-opus-4", ModelPrice{InputPer1KTokens: 0.015, OutputPer1KTokens: 0.075}},
	// Sonnet 4.6 / 4.5 / 4: $3 / $15 per 1M tokens
	{"anthropic.claude-sonnet-4-6", ModelPrice{InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015}},
	{"anthropic.claude-sonnet-4-5", ModelPrice{InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015}},
	{"anthropic.claude-sonnet-4", ModelPrice{InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015}},
	// Haiku 4.5: $1 / $5 per 1M tokens
	{"anthropic.claude-haiku-4-5", ModelPrice{InputPer1KTokens: 0.001, OutputPer1KTokens: 0.005}},
	// Haiku 3.5: $0.80 / $4 per 1M tokens
	{"anthropic.claude-haiku-3-5", ModelPrice{InputPer1KTokens: 0.0008, OutputPer1KTokens: 0.004}},
	// Sonnet 3.5: $3 / $15 per 1M tokens
	{"anthropic.claude-sonnet-3-5", ModelPrice{InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015}},
	// Claude 3 世代
	{"anthropic.claude-3-opus", ModelPrice{InputPer1KTokens: 0.015, OutputPer1KTokens: 0.075}},
	{"anthropic.claude-3-sonnet", ModelPrice{InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015}},
	{"anthropic.claude-3-haiku", ModelPrice{InputPer1KTokens: 0.00025, OutputPer1KTokens: 0.00125}},
}

func Calculate(modelID string, inputTokens, outputTokens int64) float64 {
	price, ok := lookupPrice(modelID)
	if !ok {
		return 0
	}
	return float64(inputTokens)/1000*price.InputPer1KTokens +
		float64(outputTokens)/1000*price.OutputPer1KTokens
}

func lookupPrice(modelID string) (ModelPrice, bool) {
	// inference profile ARN から モデル名部分を抽出
	// 例: "arn:aws:bedrock:us-east-1:123:inference-profile/us.anthropic.claude-opus-4-6-v1"
	//   → "anthropic.claude-opus-4-6-v1"
	normalized := NormalizeModelID(modelID)
	for _, entry := range DefaultPrices {
		if strings.Contains(normalized, entry.Prefix) {
			return entry.Price, true
		}
	}
	return ModelPrice{}, false
}

func NormalizeModelID(modelID string) string {
	// inference profile ARN の場合、最後のスラッシュ以降を取得
	if idx := strings.LastIndex(modelID, "/"); idx != -1 {
		modelID = modelID[idx+1:]
	}
	// "us." や "global." などのリージョンプレフィックスを除去
	for _, prefix := range []string{"us.", "eu.", "ap.", "global."} {
		modelID = strings.TrimPrefix(modelID, prefix)
	}
	return modelID
}
