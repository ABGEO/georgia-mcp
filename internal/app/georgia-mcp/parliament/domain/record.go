package domain

import (
	"cmp"
	"slices"

	"github.com/abgeo/georgia-mcp/internal/pkg/kernel"
)

// Bill is one draft that the Parliament reviews, with its package and its newest stage.
type Bill struct {
	ID                 ID
	Title              string
	Description        string
	Type               string
	RegistrationNumber string
	PackageName        string
	Initiator          string
	Author             string
	Registered         kernel.Date
	NewestStage        string
	// NewestReading is empty when the newest stage is not a reading.
	NewestReading Reading
}

// Stage is one review stage of a bill. Reading is empty for a committee stage. Yes and No
// are set for a stage with a vote.
type Stage struct {
	Name          string
	Reading       Reading
	From, To      *kernel.Date
	SessionDate   *kernel.Date
	BureauDate    *kernel.Date
	SessionNumber string
	BureauNumber  string
	Yes, No       *int
	Votes         []ID
}

// Vote is one plenary vote. Absent counts the members who did not vote.
type Vote struct {
	ID                 ID
	Date               kernel.Date
	BillID             ID
	BillTitle          string
	RegistrationNumber string
	Reading            Coded[Reading]
	Outcome            Coded[Outcome]
	Quorum             *int
	Yes, No, Absent    int
}

// MemberChoice is what one member did in a vote. Faction is empty for a member of no
// faction.
type MemberChoice struct {
	Member  string
	Faction string
	Choice  Coded[Choice]
}

// Question is one written question of a member to a body of the state.
type Question struct {
	ID                 int
	Type               string
	RegistrationNumber string
	Registered         kernel.Date
	Request            string
	Member             string
	Faction            string
	Addressee          string
	Deadline           *kernel.Date
	Responded          *kernel.Date
}

// FactionTally is the members of one faction in a vote and the count of each choice.
type FactionTally struct {
	Faction                string
	Yes, No, Absent, Other int
	Members                []MemberChoice
}

// Tally groups the choices by faction. The largest faction comes first, then by name.
// The members of no faction come last. The members keep their order.
func Tally(choices []MemberChoice) []FactionTally {
	var tallies []FactionTally

	index := map[string]int{}

	for _, choice := range choices {
		at, found := index[choice.Faction]
		if !found {
			at = len(tallies)
			index[choice.Faction] = at
			tallies = append(tallies, FactionTally{Faction: choice.Faction})
		}

		tallies[at].add(choice)
	}

	slices.SortStableFunc(tallies, func(left, right FactionTally) int {
		if (left.Faction == "") != (right.Faction == "") {
			return boolOrder(left.Faction == "", right.Faction == "")
		}

		return cmp.Or(
			cmp.Compare(len(right.Members), len(left.Members)),
			cmp.Compare(left.Faction, right.Faction),
		)
	})

	return tallies
}

func (f *FactionTally) add(choice MemberChoice) {
	f.Members = append(f.Members, choice)

	switch choice.Choice.Code {
	case Yes:
		f.Yes++
	case No:
		f.No++
	case Absent:
		f.Absent++
	case OtherChoice:
		f.Other++
	default:
		f.Other++
	}
}

// boolOrder puts false before true.
func boolOrder(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	default:
		return -1
	}
}
