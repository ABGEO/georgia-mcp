package tool

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// statusName gives the name of a status in the language. The ka names are those of the
// page of TAS.
func statusName(status domain.Status, language kernel.Language) string {
	names := map[domain.Status][2]string{
		domain.StatusApproved:        {"Approved", "თანხმობა"},
		domain.StatusRefused:         {"Refused", "უარყოფა"},
		domain.StatusDeficient:       {"Deficiency found", "შუალედური"},
		domain.StatusConditional:     {"Approved with conditions", "პირობადადებული"},
		domain.StatusNotReviewed:     {"Not reviewed", "განუხილველი"},
		domain.StatusRevoked:         {"Revoked", "ძალადაკარგული"},
		domain.StatusCompleted:       {"Completed", "დასრულებული"},
		domain.StatusAwaitingPayment: {"Awaiting payment", "გადახდის მოლოდინში"},
		domain.StatusInReview:        {"In review", "განხილვის პროცესში"},
	}

	pair := names[status]
	if language == kernel.LanguageGeorgian {
		return pair[1]
	}

	return pair[0]
}

func toApplicationOutput(
	application domain.Application,
	language kernel.Language,
) applicationOutput {
	output := applicationOutput{
		Number:     application.Number.String(),
		URL:        application.URL,
		Address:    application.Address,
		Registered: application.Registered.String(),
		Status: codeOutput{
			Code: string(application.Status),
			Name: statusName(application.Status, language),
		},
		Category:       application.Category,
		ParentCategory: application.ParentCategory,
		Class:          application.Class,
		Stages:         application.Stages,
		Action:         application.Action,
		Executor:       application.Executor,
	}

	if application.Deadline != nil {
		output.Deadline = application.Deadline.String()
	}

	return output
}

func toSearchOutput(list app.ApplicationList, language kernel.Language) searchOutput {
	applications := make([]applicationOutput, 0, len(list.Applications))
	for _, application := range list.Applications {
		applications = append(applications, toApplicationOutput(application, language))
	}

	return searchOutput{
		Source:       source,
		AsOf:         instant(list.ReadAt),
		Stale:        list.Stale,
		Total:        list.Total,
		Applications: applications,
	}
}

func toGetOutput(read app.ApplicationRead, language kernel.Language) getOutput {
	return getOutput{
		Source:            source,
		AsOf:              instant(read.ReadAt),
		Stale:             read.Stale,
		applicationOutput: toApplicationOutput(read.Application, language),
		DecisionText:      read.Application.DecisionText,
	}
}

func instant(at time.Time) string {
	return at.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
