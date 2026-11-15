package tool

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/permits/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// statusName gives the name of a status on the page of TAS.
func statusName(status domain.Status) string {
	names := map[domain.Status]string{
		domain.StatusApproved:        "თანხმობა",
		domain.StatusRefused:         "უარყოფა",
		domain.StatusDeficient:       "შუალედური",
		domain.StatusConditional:     "პირობადადებული",
		domain.StatusNotReviewed:     "განუხილველი",
		domain.StatusRevoked:         "ძალადაკარგული",
		domain.StatusCompleted:       "დასრულებული",
		domain.StatusAwaitingPayment: "გადახდის მოლოდინში",
		domain.StatusInReview:        "განხილვის პროცესში",
	}

	return names[status]
}

func toApplicationOutput(application domain.Application) applicationOutput {
	output := applicationOutput{
		Number:     application.Number.String(),
		URL:        application.URL,
		Address:    application.Address,
		Registered: application.Registered.String(),
		Status: codeOutput{
			Code: string(application.Status),
			Name: statusName(application.Status),
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

func toSearchOutput(list app.ApplicationList) searchOutput {
	applications := make([]applicationOutput, 0, len(list.Applications))
	for _, application := range list.Applications {
		applications = append(applications, toApplicationOutput(application))
	}

	return searchOutput{
		Source:       source,
		AsOf:         instant(list.ReadAt),
		Stale:        list.Stale,
		Total:        list.Total,
		Applications: applications,
	}
}

func toGetOutput(read app.ApplicationRead) getOutput {
	return getOutput{
		Source:            source,
		AsOf:              instant(read.ReadAt),
		Stale:             read.Stale,
		applicationOutput: toApplicationOutput(read.Application),
		DecisionText:      read.Application.DecisionText,
	}
}

func instant(at time.Time) string {
	return at.In(kernel.TbilisiZone()).Format(time.RFC3339)
}
