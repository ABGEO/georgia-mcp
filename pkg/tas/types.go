package tas

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"time"
)

// Filter selects the applications of a search. An empty field sends no filter.
type Filter struct {
	// CadastralCode is the code of a parcel: "01.18.04.020.005".
	CadastralCode string
	// DocumentNo is an application number or its prefix: "AR11168188".
	DocumentNo string
	// From and To bound the date of registration. The zero time sends no bound.
	From, To time.Time
	// StatusIDs are the codes of the statuses to return.
	StatusIDs []int
	// Start is the index of the first application, and Limit the size of the page.
	Start, Limit int
}

// Page is one page of a search.
type Page struct {
	// Total is the count of all the applications that match.
	Total int
	// Documents are the applications of the page, in the order of TAS.
	Documents []Document
}

// Document is one application. It holds no field of the applicant.
type Document struct {
	DocumentID       int64      `json:"documentId"`
	DocumentNo       string     `json:"documentNo"`
	Address          string     `json:"address"`
	RegistrationDate *Millis    `json:"registrationDate"`
	DeadLineDate     *Millis    `json:"deadLineDate"`
	DocumentStatusID int        `json:"documentStatusId"`
	CachedInfo       CachedInfo `json:"cachedInfo"`
	// ResponseText is the decision of TAS in HTML, or empty.
	ResponseText string `json:"responseText"`
}

// Millis is an instant in milliseconds since the epoch, from new Date(ms).
type Millis int64

// Time returns the instant.
func (m Millis) Time() time.Time {
	return time.UnixMilli(int64(m))
}

// CachedInfo is the summary of an application that TAS keeps as XML in a string.
type CachedInfo struct {
	StatusName   string       `xml:"documentStatusName"`
	ExecutorName string       `xml:"executorEmployeeName"`
	Nomenclature Nomenclature `xml:"nomenklature"`
}

// UnmarshalJSON reads the XML of the string. An empty string gives the zero value.
func (c *CachedInfo) UnmarshalJSON(data []byte) error {
	var text *string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("decoding the cached info: %w", err)
	}

	*c = CachedInfo{}

	if text == nil || *text == "" {
		return nil
	}

	if err := xml.Unmarshal([]byte(*text), c); err != nil {
		return fmt.Errorf("decoding the XML of the cached info: %w", err)
	}

	return nil
}

// Nomenclature is the kind of an application.
type Nomenclature struct {
	ActionName         string `xml:"actionName"`
	CategoryName       string `xml:"categoryName"`
	ParentCategoryName string `xml:"parentCategoryName"`
	// ClassName is the class of risk: "IV კლასი".
	ClassName string `xml:"className"`
	// Stadiums are the stages: "მშენებლობის ნებართვა".
	Stadiums []string `xml:"stadiums"`
}
