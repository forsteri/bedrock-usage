package query

import "fmt"

type GroupBy int

const (
	GroupByModel GroupBy = iota
	GroupByUser
	GroupByDate
)

type QueryParams struct {
	GroupBy     GroupBy
	ModelFilter string
	UserFilter  string
}

func BuildQuery(params QueryParams) string {
	base := `fields @timestamp, modelId, input.inputTokenCount as inputTokens, output.outputTokenCount as outputTokens, identity.arn`

	filters := ""
	if params.ModelFilter != "" {
		filters += fmt.Sprintf("\n| filter modelId like /%s/", params.ModelFilter)
	}
	if params.UserFilter != "" {
		filters += fmt.Sprintf("\n| filter identity.arn like /%s/", params.UserFilter)
	}

	var groupField, sortField string
	switch params.GroupBy {
	case GroupByModel:
		groupField = "modelId"
		sortField = "modelId"
	case GroupByUser:
		groupField = "identity.arn"
		sortField = "identity.arn"
	case GroupByDate:
		groupField = "datefloor(@timestamp, 1d) as date"
		sortField = "date asc"
	}

	return fmt.Sprintf(`%s%s
| stats sum(inputTokens) as totalInput, sum(outputTokens) as totalOutput, count(*) as invocations by %s
| sort %s`, base, filters, groupField, sortField)
}
