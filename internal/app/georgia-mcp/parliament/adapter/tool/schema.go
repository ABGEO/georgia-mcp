package tool

type searchBillsInput struct {
	Query  string `json:"query,omitempty"  jsonschema:"A text of the title, in Georgian, 1 to 128 characters, for example ბიუჯეტ."`
	Number string `json:"number,omitempty" jsonschema:"A registration number, for example 372/2-XIმპ."`
	From   string `json:"from,omitempty"   jsonschema:"The first date of registration, YYYY-MM-DD."`
	To     string `json:"to,omitempty"     jsonschema:"The last date of registration, YYYY-MM-DD."`
	Page   int    `json:"page,omitempty"   jsonschema:"The page, from 1. The default is 1."`
	Limit  int    `json:"limit,omitempty"  jsonschema:"The count of bills of a page, 1 to 50. The default is 10."`
}

type getInput struct {
	ID int `json:"id" jsonschema:"The identifier, from a search or a list, for example 32449."`
}

type listVotesInput struct {
	Query string `json:"query,omitempty" jsonschema:"A text of the title of the bill, in Georgian, 1 to 128 characters."`
	From  string `json:"from,omitempty"  jsonschema:"The first date of the vote, YYYY-MM-DD."`
	To    string `json:"to,omitempty"    jsonschema:"The last date of the vote, YYYY-MM-DD."`
	Page  int    `json:"page,omitempty"  jsonschema:"The page, from 1. The default is 1."`
	Limit int    `json:"limit,omitempty" jsonschema:"The count of votes of a page, 1 to 50. The default is 10."`
}

type searchQuestionsInput struct {
	Query  string `json:"query,omitempty"  jsonschema:"A text of the request, in Georgian, 1 to 128 characters."`
	Member string `json:"member,omitempty" jsonschema:"A name of the member, in Georgian: the family name, or the family name and the first name."`
	From   string `json:"from,omitempty"   jsonschema:"The first date of registration, YYYY-MM-DD."`
	To     string `json:"to,omitempty"     jsonschema:"The last date of registration, YYYY-MM-DD."`
	Page   int    `json:"page,omitempty"   jsonschema:"The page, from 1. The default is 1."`
	Limit  int    `json:"limit,omitempty"  jsonschema:"The count of questions of a page, 1 to 50. The default is 10."`
}

// meta opens each output.
type meta struct {
	Source string `json:"source" jsonschema:"The source of the data: Parliament of Georgia."`
	AsOf   string `json:"as_of"  jsonschema:"The instant at which the server read the data, RFC 3339."`
	Stale  bool   `json:"stale"  jsonschema:"True when the Parliament did not answer and the server returns data it read before."`
}

// pageMeta follows meta in a search or a list.
type pageMeta struct {
	Total  int    `json:"total"            jsonschema:"The count of all the matches at the source."`
	Page   int    `json:"page"             jsonschema:"The page of this answer, from 1."`
	Notice string `json:"notice,omitempty" jsonschema:"Set when a text with no Georgian letter found nothing."`
}

type codeOutput struct {
	Code string `json:"code" jsonschema:"The code."`
	Name string `json:"name" jsonschema:"The name at the Parliament, in Georgian."`
}

type billsOutput struct {
	meta
	pageMeta

	Bills []billSummary `json:"bills" jsonschema:"The bills of the page, newest first."`
}

type billSummary struct {
	ID                 int    `json:"id"                       jsonschema:"The bill identifier."`
	Title              string `json:"title"                    jsonschema:"The title of the bill."`
	Type               string `json:"type"                     jsonschema:"The bill type."`
	RegistrationNumber string `json:"registration_number"      jsonschema:"The registration number of the package."`
	Registered         string `json:"registered"               jsonschema:"The date of registration, YYYY-MM-DD."`
	Initiator          string `json:"initiator"                jsonschema:"The body that submitted the bill."`
	Author             string `json:"author,omitempty"         jsonschema:"The body that drafted the bill."`
	NewestStage        string `json:"newest_stage"             jsonschema:"The name of the newest review stage."`
	NewestReading      string `json:"newest_reading,omitempty" jsonschema:"The reading code of the newest stage, when it is a reading."`
	URL                string `json:"url"                      jsonschema:"The page of the bill at the Parliament."`
}

type billOutput struct {
	meta
	billSummary

	Description string        `json:"description,omitempty" jsonschema:"The description of the bill."`
	PackageName string        `json:"package_name"          jsonschema:"The name of the package of the bill."`
	Stages      []stageOutput `json:"stages"                jsonschema:"Every review stage, in the order of the review."`
}

type stageOutput struct {
	Name          string `json:"name"                     jsonschema:"The name of the stage at the Parliament."`
	Reading       string `json:"reading,omitempty"        jsonschema:"first_reading, second_reading, third_reading, single_reading, plenary_draft, alternative_proposal, or other. Empty for a committee stage."`
	From          string `json:"from,omitempty"           jsonschema:"The first day of the review, YYYY-MM-DD."`
	To            string `json:"to,omitempty"             jsonschema:"The last day of the review, YYYY-MM-DD."`
	SessionDate   string `json:"session_date,omitempty"   jsonschema:"The day of the plenary sitting, YYYY-MM-DD."`
	SessionNumber string `json:"session_number,omitempty" jsonschema:"The number of the plenary sitting."`
	BureauDate    string `json:"bureau_date,omitempty"    jsonschema:"The day of the sitting of the bureau, YYYY-MM-DD."`
	BureauNumber  string `json:"bureau_number,omitempty"  jsonschema:"The number of the sitting of the bureau."`
	Yes           *int   `json:"yes,omitempty"            jsonschema:"The count of yes."`
	No            *int   `json:"no,omitempty"             jsonschema:"The count of no."`
	Votes         []int  `json:"votes,omitempty"          jsonschema:"The identifiers of the votes of the stage."`
}

type votesOutput struct {
	meta
	pageMeta

	Votes []voteSummary `json:"votes" jsonschema:"The votes of the page, newest first."`
}

type voteSummary struct {
	ID                 int        `json:"id"                  jsonschema:"The vote identifier."`
	Date               string     `json:"date"                jsonschema:"The day of the vote, YYYY-MM-DD."`
	BillID             int        `json:"bill_id"             jsonschema:"The bill identifier."`
	BillTitle          string     `json:"bill_title"          jsonschema:"The title of the bill."`
	RegistrationNumber string     `json:"registration_number" jsonschema:"The registration number of the package."`
	Reading            codeOutput `json:"reading"             jsonschema:"The reading: first_reading, second_reading, third_reading, single_reading, plenary_draft, alternative_proposal, or other."`
	Outcome            codeOutput `json:"outcome"             jsonschema:"adopted, rejected, or other."`
	Quorum             *int       `json:"quorum,omitempty"    jsonschema:"The count of yes that the item needs."`
	Yes                int        `json:"yes"                 jsonschema:"The count of yes."`
	No                 int        `json:"no"                  jsonschema:"The count of no."`
	Absent             int        `json:"absent"              jsonschema:"The count of members who did not vote."`
	URL                string     `json:"url"                 jsonschema:"The page of the bill at the Parliament."`
}

type voteOutput struct {
	meta
	voteSummary

	Factions []factionOutput `json:"factions" jsonschema:"The members by faction, the largest first. A faction with an empty name holds the members of no faction."`
}

type factionOutput struct {
	Name    string         `json:"name"    jsonschema:"The faction, in Georgian. Empty for the members of no faction."`
	Yes     int            `json:"yes"     jsonschema:"The count of yes."`
	No      int            `json:"no"      jsonschema:"The count of no."`
	Absent  int            `json:"absent"  jsonschema:"The count of absent."`
	Other   int            `json:"other"   jsonschema:"The count of any other choice."`
	Members []memberOutput `json:"members" jsonschema:"The members of the faction, by family name."`
}

type memberOutput struct {
	Name   string `json:"name"   jsonschema:"The family name and the first name."`
	Choice string `json:"choice" jsonschema:"yes, no, absent, or other."`
}

type questionsOutput struct {
	meta
	pageMeta

	Questions []questionOutput `json:"questions" jsonschema:"The questions of the page, newest first."`
}

type questionOutput struct {
	ID                 int    `json:"id"                  jsonschema:"The question identifier."`
	Type               string `json:"type"                jsonschema:"The type of the question."`
	RegistrationNumber string `json:"registration_number" jsonschema:"The registration number."`
	Registered         string `json:"registered"          jsonschema:"The date of registration, YYYY-MM-DD."`
	Request            string `json:"request"             jsonschema:"The request, as the Parliament summarizes it."`
	Member             string `json:"member"              jsonschema:"The member who asked."`
	Faction            string `json:"faction,omitempty"   jsonschema:"The faction of the member."`
	Addressee          string `json:"addressee"           jsonschema:"The body and the person that the question goes to."`
	Deadline           string `json:"deadline,omitempty"  jsonschema:"The last day of the answer, YYYY-MM-DD."`
	Responded          string `json:"responded,omitempty" jsonschema:"The day of the answer, YYYY-MM-DD."`
	Status             string `json:"status"              jsonschema:"answered, pending, or overdue."`
	URL                string `json:"url"                 jsonschema:"The page of the question, with the files of the question and the answer."`
}
