package tool

type searchInput struct {
	Code  string `json:"code,omitempty"  jsonschema:"The cadastral code of the parcel: 4 or 5 groups of digits with dots, as 01.18.04.020.005."`
	From  string `json:"from,omitempty"  jsonschema:"The first date of registration, YYYY-MM-DD. Without to, the period is 7 days."`
	To    string `json:"to,omitempty"    jsonschema:"The last date of registration, YYYY-MM-DD. Without from, the period is 7 days."`
	Limit int    `json:"limit,omitempty" jsonschema:"The maximum count of applications, 1 to 50. The default is 10."`
}

type getInput struct {
	Number string `json:"number" jsonschema:"The application number, as AR11168188."`
}

type searchOutput struct {
	Source       string              `json:"source"       jsonschema:"The source of the data: TAS."`
	AsOf         string              `json:"as_of"        jsonschema:"The instant at which the server read the applications, RFC 3339."`
	Stale        bool                `json:"stale"        jsonschema:"True when TAS did not answer and the server returns applications it read before."`
	Total        int                 `json:"total"        jsonschema:"The count of all the applications that match."`
	Applications []applicationOutput `json:"applications" jsonschema:"The applications, newest first."`
}

type getOutput struct {
	applicationOutput

	Source       string `json:"source"                  jsonschema:"The source of the data: TAS."`
	AsOf         string `json:"as_of"                   jsonschema:"The instant at which the server read the application, RFC 3339."`
	Stale        bool   `json:"stale"                   jsonschema:"True when TAS did not answer and the server returns an application it read before."`
	DecisionText string `json:"decision_text,omitempty" jsonschema:"The text of the decision of TAS, in Georgian, with no HTML. Empty before a decision."`
}

type applicationOutput struct {
	Number         string     `json:"number"                    jsonschema:"The application number."`
	URL            string     `json:"url"                       jsonschema:"The page of the application on tas.ge."`
	Address        string     `json:"address,omitempty"         jsonschema:"The address, in Georgian, as TAS publishes it."`
	Registered     string     `json:"registered"                jsonschema:"The date of registration, YYYY-MM-DD."`
	Deadline       string     `json:"deadline,omitempty"        jsonschema:"The date by which TAS must answer, YYYY-MM-DD."`
	Status         codeOutput `json:"status"                    jsonschema:"approved, refused, deficient, conditional, not_reviewed, revoked, completed, awaiting_payment, or in_review."`
	Category       string     `json:"category,omitempty"        jsonschema:"The kind of object or work, in Georgian."`
	ParentCategory string     `json:"parent_category,omitempty" jsonschema:"The group of the category, in Georgian."`
	Class          string     `json:"class,omitempty"           jsonschema:"The class of risk, I to IV."`
	Stages         []string   `json:"stages,omitempty"          jsonschema:"The stages that the application covers, in Georgian."`
	Action         string     `json:"action,omitempty"          jsonschema:"The action, in Georgian, as a change of an approved project."`
	Executor       string     `json:"executor,omitempty"        jsonschema:"The employee of TAS who handles the application."`
}

type codeOutput struct {
	Code string `json:"code" jsonschema:"The code."`
	Name string `json:"name" jsonschema:"The name of the code at TAS, in Georgian."`
}
