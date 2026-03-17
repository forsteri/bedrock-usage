package pricing

import "strings"

type ModelPrice struct {
	InputPer1KTokens  float64
	OutputPer1KTokens float64
}

var DefaultPrices = map[string]ModelPrice{
	"anthropic.claude-sonnet-4":   {InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015},
	"anthropic.claude-opus-4":     {InputPer1KTokens: 0.015, OutputPer1KTokens: 0.075},
	"anthropic.claude-haiku-4-5":  {InputPer1KTokens: 0.0008, OutputPer1KTokens: 0.004},
	"anthropic.claude-haiku-3-5":  {InputPer1KTokens: 0.0008, OutputPer1KTokens: 0.004},
	"anthropic.claude-sonnet-3-5": {InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015},
	"anthropic.claude-3-sonnet":   {InputPer1KTokens: 0.003, OutputPer1KTokens: 0.015},
	"anthropic.claude-3-haiku":    {InputPer1KTokens: 0.00025, OutputPer1KTokens: 0.00125},
	"anthropic.claude-3-opus":     {InputPer1KTokens: 0.015, OutputPer1KTokens: 0.075},
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
	for prefix, price := range DefaultPrices {
		if strings.Contains(normalized, prefix) {
			return price, true
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
