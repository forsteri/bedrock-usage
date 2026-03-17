package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/forsteri/bedrock-usage/internal/display"
	"github.com/forsteri/bedrock-usage/internal/query"
	"github.com/spf13/cobra"
)

var (
	period   string
	daily    bool
	model    string
	user     string
	byUser   bool
	logGroup string

	version = "dev"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "bedrock-usage",
		Short:   "Bedrock のトークン使用量と予想利用料金を表示するCLIツール",
		Version: version,
		RunE:    run,
	}

	rootCmd.Flags().StringVarP(&period, "period", "p", "today", "期間指定 (today, week, month)")
	rootCmd.Flags().BoolVarP(&daily, "daily", "d", false, "日次推移を表示")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "モデルIDでフィルタ (部分一致)")
	rootCmd.Flags().StringVarP(&user, "user", "u", "", "IAMユーザー/ロールでフィルタ (部分一致)")
	rootCmd.Flags().BoolVar(&byUser, "by-user", false, "ユーザー別の内訳を表示")
	rootCmd.Flags().StringVarP(&logGroup, "log-group", "l", "bedrock/modelinvocations", "CloudWatch Logsのロググループ名")

	return rootCmd
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	startTime, endTime, err := resolvePeriod(period)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("AWS設定の読み込みに失敗しました: %w", err)
	}

	cwlClient := cloudwatchlogs.NewFromConfig(cfg)
	client := query.NewClient(cwlClient, logGroup)

	periodLabel := periodLabel(period)

	if daily {
		return runDaily(ctx, client, startTime, endTime, periodLabel)
	}
	if byUser {
		return runByUser(ctx, client, startTime, endTime, periodLabel)
	}
	return runSummary(ctx, client, startTime, endTime, periodLabel)
}

func runSummary(ctx context.Context, client *query.Client, start, end time.Time, periodLabel string) error {
	records, err := client.Query(ctx, start, end, query.QueryParams{
		GroupBy:     query.GroupByModel,
		ModelFilter: model,
		UserFilter:  user,
	})
	if err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("\n指定期間にデータがありません。")
		fmt.Println("Model Invocation Logging が有効になっているか確認してください。")
		return nil
	}

	title := fmt.Sprintf("Bedrock Usage Summary (%s)", periodLabel)
	if user != "" {
		title += fmt.Sprintf(" | User: %s", user)
	}
	display.RenderSummary(records, title, "Model")
	return nil
}

func runByUser(ctx context.Context, client *query.Client, start, end time.Time, periodLabel string) error {
	records, err := client.Query(ctx, start, end, query.QueryParams{
		GroupBy:     query.GroupByUser,
		ModelFilter: model,
		UserFilter:  user,
	})
	if err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("\n指定期間にデータがありません。")
		return nil
	}

	title := fmt.Sprintf("Bedrock Usage by User (%s)", periodLabel)
	display.RenderSummary(records, title, "User")
	return nil
}

func runDaily(ctx context.Context, client *query.Client, start, end time.Time, periodLabel string) error {
	records, err := client.Query(ctx, start, end, query.QueryParams{
		GroupBy:     query.GroupByDate,
		ModelFilter: model,
		UserFilter:  user,
	})
	if err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("\n指定期間にデータがありません。")
		return nil
	}

	title := fmt.Sprintf("Daily Breakdown (%s)", periodLabel)
	display.RenderDaily(records, title, model)
	return nil
}

func resolvePeriod(p string) (time.Time, time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := now

	switch p {
	case "today":
		return today, end, nil
	case "week":
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := today.AddDate(0, 0, -(weekday - 1))
		return monday, end, nil
	case "month":
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return firstOfMonth, end, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("不正な期間指定です: %s (today, week, month のいずれかを指定してください)", p)
	}
}

func periodLabel(p string) string {
	now := time.Now()
	switch p {
	case "today":
		return now.Format("2006-01-02")
	case "week":
		start, _, _ := resolvePeriod("week")
		return fmt.Sprintf("%s ~ %s", start.Format("2006-01-02"), now.Format("2006-01-02"))
	case "month":
		return now.Format("2006-01")
	default:
		return p
	}
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
