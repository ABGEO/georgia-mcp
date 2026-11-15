package tool

import (
	"time"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/app"
	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/parliament/domain"
	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

const source = "Parliament of Georgia"

func toMeta(readAt time.Time, stale bool) meta {
	return meta{
		Source: source,
		AsOf:   readAt.In(kernel.TbilisiZone()).Format(time.RFC3339),
		Stale:  stale,
	}
}

func dateText(date *kernel.Date) string {
	if date == nil {
		return ""
	}

	return date.String()
}

func toBillSummary(bill domain.Bill) billSummary {
	return billSummary{
		ID:                 int(bill.ID),
		Title:              bill.Title,
		Type:               bill.Type,
		RegistrationNumber: bill.RegistrationNumber,
		Registered:         bill.Registered.String(),
		Initiator:          bill.Initiator,
		Author:             bill.Author,
		NewestStage:        bill.NewestStage,
		NewestReading:      string(bill.NewestReading),
		URL:                bill.URL,
	}
}

func toBillsOutput(result app.BillResult, page domain.Page) billsOutput {
	bills := make([]billSummary, 0, len(result.Bills))
	for _, bill := range result.Bills {
		bills = append(bills, toBillSummary(bill))
	}

	return billsOutput{
		meta:  toMeta(result.ReadAt, result.Stale),
		Total: result.Total, Page: page.Number, Notice: result.Notice,
		Bills: bills,
	}
}

func toBillOutput(read app.BillRead) billOutput {
	stages := make([]stageOutput, 0, len(read.Stages))

	for _, stage := range read.Stages {
		votes := make([]int, 0, len(stage.Votes))
		for _, id := range stage.Votes {
			votes = append(votes, int(id))
		}

		stages = append(stages, stageOutput{
			Name:          stage.Name,
			Reading:       string(stage.Reading),
			From:          dateText(stage.From),
			To:            dateText(stage.To),
			SessionDate:   dateText(stage.SessionDate),
			SessionNumber: stage.SessionNumber,
			BureauDate:    dateText(stage.BureauDate),
			BureauNumber:  stage.BureauNumber,
			Yes:           stage.Yes,
			No:            stage.No,
			Votes:         votes,
		})
	}

	return billOutput{
		meta:        toMeta(read.ReadAt, read.Stale),
		billSummary: toBillSummary(read.Bill),
		Description: read.Bill.Description,
		PackageName: read.Bill.PackageName,
		Stages:      stages,
	}
}

func toVoteSummary(vote domain.Vote) voteSummary {
	return voteSummary{
		ID:                 int(vote.ID),
		Date:               vote.Date.String(),
		BillID:             int(vote.BillID),
		BillTitle:          vote.BillTitle,
		RegistrationNumber: vote.RegistrationNumber,
		Reading:            codeOutput{Code: string(vote.Reading.Code), Name: vote.Reading.Name},
		Outcome:            codeOutput{Code: string(vote.Outcome.Code), Name: vote.Outcome.Name},
		Quorum:             vote.Quorum,
		Yes:                vote.Yes,
		No:                 vote.No,
		Absent:             vote.Absent,
		URL:                vote.URL,
	}
}

func toVotesOutput(result app.VoteResult, page domain.Page) votesOutput {
	votes := make([]voteSummary, 0, len(result.Votes))
	for _, vote := range result.Votes {
		votes = append(votes, toVoteSummary(vote))
	}

	return votesOutput{
		meta:  toMeta(result.ReadAt, result.Stale),
		Total: result.Total, Page: page.Number, Notice: result.Notice,
		Votes: votes,
	}
}

func toVoteOutput(detail app.VoteDetail) voteOutput {
	factions := make([]factionOutput, 0, len(detail.Factions))

	for _, tally := range detail.Factions {
		members := make([]memberOutput, 0, len(tally.Members))
		for _, member := range tally.Members {
			members = append(members, memberOutput{
				Name: member.Member, Choice: string(member.Choice.Code),
			})
		}

		factions = append(factions, factionOutput{
			Name:    tally.Faction,
			Yes:     tally.Yes,
			No:      tally.No,
			Absent:  tally.Absent,
			Other:   tally.Other,
			Members: members,
		})
	}

	return voteOutput{
		meta:        toMeta(detail.ReadAt, detail.Stale),
		voteSummary: toVoteSummary(detail.Vote),
		Factions:    factions,
	}
}

func toQuestionsOutput(result app.QuestionResult, page domain.Page) questionsOutput {
	questions := make([]questionOutput, 0, len(result.Questions))

	for _, item := range result.Questions {
		question := item.Question
		questions = append(questions, questionOutput{
			ID:                 question.ID,
			Type:               question.Type,
			RegistrationNumber: question.RegistrationNumber,
			Registered:         question.Registered.String(),
			Request:            question.Request,
			Member:             question.Member,
			Faction:            question.Faction,
			Addressee:          question.Addressee,
			Deadline:           dateText(question.Deadline),
			Responded:          dateText(question.Responded),
			Status:             string(item.Status),
			URL:                question.URL,
		})
	}

	return questionsOutput{
		meta:  toMeta(result.ReadAt, result.Stale),
		Total: result.Total, Page: page.Number, Notice: result.Notice,
		Questions: questions,
	}
}
