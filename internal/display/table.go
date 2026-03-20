package display

import (
	"fmt"
	"os"
	"strings"

	"github.com/forsteri/bedrock-usage/internal/pricing"
	"github.com/forsteri/bedrock-usage/internal/query"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

func RenderSummary(records []query.UsageRecord, title string, keyHeader string) {
	fmt.Println()
	fmt.Println(title)
	fmt.Println()

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignLeft}},
			Row:    tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignRight}},
			Footer: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignRight}},
		}),
	)

	table.Header([]string{keyHeader, "Input Tokens", "Output Tokens", "Invocations", "Est. Cost"})

	var totalInput, totalOutput, totalInvocations int64
	var totalCost float64

	for _, rec := range records {
		cost := pricing.Calculate(rec.Key, rec.InputTokens, rec.OutputTokens)
		totalCost += cost
		totalInput += rec.InputTokens
		totalOutput += rec.OutputTokens
		totalInvocations += rec.Invocations

		table.Append([]string{
			shortenKey(keyHeader, rec.Key),
			formatInt(rec.InputTokens),
			formatInt(rec.OutputTokens),
			formatInt(rec.Invocations),
			formatCost(cost),
		})
	}

	table.Footer([]string{
		"Total",
		formatInt(totalInput),
		formatInt(totalOutput),
		formatInt(totalInvocations),
		formatCost(totalCost),
	})

	table.Render()
	fmt.Println()
}

func RenderDaily(records []query.UsageRecord, title string, modelFilter string) {
	costFunc := func(rec query.UsageRecord) float64 {
		if modelFilter != "" {
			return pricing.Calculate(modelFilter, rec.InputTokens, rec.OutputTokens)
		}
		return 0
	}

	fmt.Println()
	fmt.Println(title)
	fmt.Println()

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignLeft}},
			Row:    tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignRight}},
			Footer: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignRight}},
		}),
	)

	table.Header([]string{"Date", "Input Tokens", "Output Tokens", "Invocations", "Est. Cost"})

	var totalInput, totalOutput, totalInvocations int64
	var totalCost float64

	for _, rec := range records {
		cost := costFunc(rec)
		totalCost += cost
		totalInput += rec.InputTokens
		totalOutput += rec.OutputTokens
		totalInvocations += rec.Invocations

		table.Append([]string{
			rec.Key,
			formatInt(rec.InputTokens),
			formatInt(rec.OutputTokens),
			formatInt(rec.Invocations),
			formatCost(cost),
		})
	}

	table.Footer([]string{
		"Total",
		formatInt(totalInput),
		formatInt(totalOutput),
		formatInt(totalInvocations),
		formatCost(totalCost),
	})

	table.Render()
	fmt.Println()
}

func RenderPrices() {
	fmt.Println()
	fmt.Println("Built-in Pricing Table (per 1K tokens)")
	fmt.Println()

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignLeft}},
			Row:    tw.CellConfig{Alignment: tw.CellAlignment{Global: tw.AlignRight}},
		}),
	)

	table.Header([]string{"Match Key", "Input", "Output"})

	for _, entry := range pricing.DefaultPrices {
		table.Append([]string{
			entry.Prefix,
			fmt.Sprintf("$%.4f", entry.Price.InputPer1KTokens),
			fmt.Sprintf("$%.5f", entry.Price.OutputPer1KTokens),
		})
	}

	table.Render()
	fmt.Println()
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	result := make([]byte, 0, len(s)+(len(s)-1)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func formatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

func shortenKey(keyHeader, key string) string {
	switch keyHeader {
	case "Model":
		return pricing.NormalizeModelID(key)
	case "User":
		return shortenARN(key)
	default:
		return key
	}
}

func shortenARN(arn string) string {
	// "arn:aws:sts::123:assumed-role/RoleName/user@example.com"
	//  → "RoleName/user@example.com"
	// "arn:aws:iam::123:user/alice" → "user/alice"
	parts := strings.SplitN(arn, "/", 2)
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]
		if strings.Contains(prefix, "assumed-role") {
			return suffix
		}
		if strings.Contains(prefix, ":user") {
			return "user/" + suffix
		}
	}
	return arn
}
