package parliament

// A date of a query or an answer is dd-MM-yyyy: "30-09-2026". An empty field of a query is
// not sent. Start and Limit are always sent: the Parliament refuses a list call without
// them.

// BillQuery selects the bills. The Parliament gives at most 50 on a page.
type BillQuery struct {
	Start, Limit int
	// BillName is a text of the title.
	BillName string
	// RegistrationNumber is the number of a package: "372/2-XIმპ".
	RegistrationNumber string
	// RegistrationDate1 and RegistrationDate2 bound the date of registration.
	RegistrationDate1, RegistrationDate2 string
}

// VoteSessionQuery selects the plenary votes.
type VoteSessionQuery struct {
	Start, Limit int
	// BillRd is a text of the title of the bill.
	BillRd string
	// BillReviewID is a review stage of a bill. 0 is not sent.
	BillReviewID int
	// VotingDate1 and VotingDate2 bound the date of the vote.
	VotingDate1, VotingDate2 string
}

// MPQuestionQuery selects the member questions.
type MPQuestionQuery struct {
	Start, Limit int
	// RequestTitle is a text of the request.
	RequestTitle string
	// ColleagueName is a name of the member: "ხორგუანი".
	ColleagueName string
	// RegistrationDate1 and RegistrationDate2 bound the date of registration.
	RegistrationDate1, RegistrationDate2 string
}

// Library is an item of a list of the Parliament: a type, a result, a conclusion.
type Library struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BillList is one page of bills and the count of every match.
type BillList struct {
	Total int    `json:"total"`
	List  []Bill `json:"list"`
}

// Bill is one draft that the Parliament reviews.
type Bill struct {
	ID              int         `json:"id"`
	BillName        string      `json:"billName"`
	BillDescription *string     `json:"billDescription"`
	BillPackage     BillPackage `json:"billPackage"`
	BillType        Library     `json:"billType"`
	// BillReviewTypes holds the review stages that the bill passed, in the order of the
	// review.
	BillReviewTypes []Library `json:"billReviewTypes"`
}

// BillPackage is the package that holds a bill.
type BillPackage struct {
	ID                 int    `json:"id"`
	RegistrationNumber string `json:"registrationNumber"`
	PackageName        string `json:"packageName"`
	Initiator          string `json:"initiator"`
	Author             string `json:"author"`
	RegistrationDate   string `json:"registrationDate"`
	BillCount          int    `json:"billCount"`
}

// BillReview is one review stage of a bill. VoteY and VoteN are set for a stage with a
// vote.
type BillReview struct {
	ID                 int     `json:"id"`
	Index              int     `json:"index"`
	ReviewTypeName     string  `json:"reviewTypeName"`
	SessionNumber      *string `json:"sessionNumber"`
	BureauNumber       *string `json:"bureauNumber"`
	SessionReviewDate  *string `json:"sessionReviewDate"`
	BureauReviewDate   *string `json:"bureauReviewDate"`
	BillReviewFromDate *string `json:"billReviewFromDate"`
	BillReviewToDate   *string `json:"billReviewToDate"`
	VoteY              *int    `json:"voteY"`
	VoteN              *int    `json:"voteN"`
}

// VoteSessionList is one page of plenary votes and the count of every match.
type VoteSessionList struct {
	Total int           `json:"total"`
	List  []VoteSession `json:"list"`
}

// VoteSession is one plenary vote. A field that ends in Rd describes the bill.
type VoteSession struct {
	ID                   int             `json:"id"`
	Quorum               *int            `json:"quorum"`
	VotingConclusionType *Library        `json:"votingConclusionType"`
	ImportConferenceDate string          `json:"importConferenceDate"`
	BillReviewTypeRd     string          `json:"billReviewTypeRd"`
	BillIDRd             int             `json:"billIdRd"`
	BillRd               string          `json:"billRd"`
	RegistrationNumberRd string          `json:"registrationNumberRd"`
	VotingResultSet      VotingResultSet `json:"votingResultSet"`
}

// VotingResultSet holds the counts of a vote. WithoutInteraction counts the members who
// did not vote.
type VotingResultSet struct {
	Yes                int `json:"yes"`
	No                 int `json:"no"`
	WithoutInteraction int `json:"withoutInteraction"`
}

// VotingList is the choices of the members in one vote.
type VotingList struct {
	Total int      `json:"total"`
	List  []Voting `json:"list"`
}

// Voting is the choice of one member. FactionRd is nil for a member of no faction.
type Voting struct {
	VotingResult *Library `json:"votingResult"`
	ColleagueRd  string   `json:"colleagueRd"`
	FactionRd    *string  `json:"factionRd"`
}

// MPQuestionList is one page of member questions and the count of every match.
type MPQuestionList struct {
	Total int          `json:"total"`
	List  []MPQuestion `json:"list"`
}

// MPQuestion is one written question of a member to a body of the state.
type MPQuestion struct {
	ID                 int      `json:"id"`
	Type               *Library `json:"type"`
	RegistrationNumber string   `json:"registrationNumber"`
	RegistrationDate   string   `json:"registrationDate"`
	Request            string   `json:"request"`
	Target             *string  `json:"target"`
	TargetOrganisation *string  `json:"targetOrganisation"`
	TargetPerson       *string  `json:"targetPerson"`
	// Deadline1 is the last day of the answer.
	Deadline1     *string `json:"deadline1"`
	ResponseDate  *string `json:"responseDate"`
	ColleagueName *string `json:"colleagueName"`
	// NodeName is the faction of the member.
	NodeName *string `json:"nodeName"`
}
