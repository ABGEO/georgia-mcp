package tool

type searchActsInput struct {
	Query      string `json:"query"                 jsonschema:"The text to find, 1 to 128 characters. Georgian finds every act; English finds translated acts only."`
	Target     string `json:"target,omitempty"      jsonschema:"Where to search: title or text. The default is title."`
	Type       string `json:"type,omitempty"        jsonschema:"A document type: law, parliament_resolution, presidential_decree, government_resolution, government_decree, edict, order, international_treaty, constitutional_court_decision, court_act, self_government_ordinance, self_government_decision, revenue_service_order, individual_legal_act, eu_law, or draft. A code is a law."`
	Status     string `json:"status,omitempty"      jsonschema:"The status of the act: active, expired, or pending."`
	Issuer     string `json:"issuer,omitempty"      jsonschema:"The body that adopted the act, in the language of the request, for example Parliament of Georgia."`
	Number     string `json:"number,omitempty"      jsonschema:"The number of the act, for example 3591."`
	SignedFrom string `json:"signed_from,omitempty" jsonschema:"The first signing date, YYYY-MM-DD."`
	SignedTo   string `json:"signed_to,omitempty"   jsonschema:"The last signing date, YYYY-MM-DD."`
	Order      string `json:"order,omitempty"       jsonschema:"newest or oldest signing date first. The default is newest."`
	Page       int    `json:"page,omitempty"        jsonschema:"The page of 20 results, from 1. The default is 1."`
	Language   string `json:"language,omitempty"    jsonschema:"The language of the titles: ka or en. The default is en."`
	Limit      int    `json:"limit,omitempty"       jsonschema:"The maximum count of acts of the page, 1 to 20. The default is 20."`
}

type getActInput struct {
	ID       string   `json:"id"                 jsonschema:"The identifier of the act from law_search_acts, for example 1043717."`
	Version  *int     `json:"version,omitempty"  jsonschema:"A version number from the versions of the act, from 0. Give a version or a date, not both."`
	Date     string   `json:"date,omitempty"     jsonschema:"A date, YYYY-MM-DD. Reads the newest version published on or before it."`
	Articles []string `json:"articles,omitempty" jsonschema:"At most 10 article numbers, for example 82 or 133^3. Leave empty for the table of contents."`
	Piece    int      `json:"piece,omitempty"    jsonschema:"For an act with no articles: the piece of 50,000 characters of its text, from 1. The default is 1."`
	Language string   `json:"language,omitempty" jsonschema:"The language of the text: ka or en. The default is en. Only ka is official."`
}

type searchOutput struct {
	Source string       `json:"source" jsonschema:"The source of the data: Legislative Herald of Georgia."`
	AsOf   string       `json:"as_of"  jsonschema:"The instant at which the server read the results, RFC 3339."`
	Stale  bool         `json:"stale"  jsonschema:"True when the source did not answer and the server returns results it read before."`
	Total  int          `json:"total"  jsonschema:"The count of matching acts at the source."`
	Page   int          `json:"page"   jsonschema:"The page of 20 results."`
	Acts   []actSummary `json:"acts"   jsonschema:"The acts of the page."`
}

type actSummary struct {
	ID     string `json:"id"               jsonschema:"The identifier of the act."`
	Title  string `json:"title"            jsonschema:"The title of the act."`
	Type   string `json:"type,omitempty"   jsonschema:"The document type as the source names it."`
	Issuer string `json:"issuer,omitempty" jsonschema:"The body that adopted the act."`
	Number string `json:"number,omitempty" jsonschema:"The number of the act."`
	Signed string `json:"signed,omitempty" jsonschema:"The signing date, YYYY-MM-DD."`
	Status string `json:"status,omitempty" jsonschema:"active, expired, or pending."`
	URL    string `json:"url"              jsonschema:"The page of the act at the Legislative Herald."`
}

type actOutput struct {
	Source        string          `json:"source"                      jsonschema:"The source of the data: Legislative Herald of Georgia."`
	AsOf          string          `json:"as_of"                       jsonschema:"The instant at which the server read the act, RFC 3339."`
	Stale         bool            `json:"stale"                       jsonschema:"True when the source did not answer and the server returns a text it read before."`
	URL           string          `json:"url"                         jsonschema:"The page of this version at the Legislative Herald."`
	Language      string          `json:"language"                    jsonschema:"ka or en."`
	Official      bool            `json:"official"                    jsonschema:"True for the Georgian text. False for a translation."`
	Notice        string          `json:"notice,omitempty"            jsonschema:"Set for a translation: only the Georgian text is official."`
	ID            string          `json:"id"                          jsonschema:"The identifier of the act."`
	Title         string          `json:"title"                       jsonschema:"The title of the act."`
	Type          string          `json:"type,omitempty"              jsonschema:"The document type as the source names it."`
	Issuer        string          `json:"issuer,omitempty"            jsonschema:"The body that adopted the act."`
	Number        string          `json:"number,omitempty"            jsonschema:"The number of the act."`
	Signed        string          `json:"signed,omitempty"            jsonschema:"The signing date, YYYY-MM-DD."`
	Status        string          `json:"status,omitempty"            jsonschema:"active, expired, or pending. Empty when the source gives none."`
	Registration  string          `json:"registration_code,omitempty" jsonschema:"The registration code of the act."`
	Version       versionOutput   `json:"version"                     jsonschema:"The version of this text."`
	NewestVersion *versionOutput  `json:"newest_version,omitempty"    jsonschema:"The newest version of the act, when it is newer than this text. A translation can trail the Georgian text."`
	Versions      []versionOutput `json:"versions,omitempty"          jsonschema:"The newest 20 versions, newest first. Only when no article is requested."`
	VersionCount  int             `json:"version_count,omitempty"     jsonschema:"The count of every version of the act. Give a version or a date to read an older one."`
	Contents      []headingOutput `json:"contents,omitempty"          jsonschema:"The sections, chapters, and articles, in order. Only when no article is requested."`
	Articles      []articleOutput `json:"articles,omitempty"          jsonschema:"The requested articles, in the order of the act."`
	Next          string          `json:"next_article,omitempty"      jsonschema:"The first requested article left out by the size limit. Ask for it in a new call."`
	Truncated     bool            `json:"truncated,omitempty"         jsonschema:"True when one article passed the size limit and its last paragraphs are left out."`
	Text          string          `json:"text,omitempty"              jsonschema:"The text of an act that has no articles: one piece."`
	Piece         int             `json:"piece,omitempty"             jsonschema:"The piece of the text, from 1."`
	Pieces        int             `json:"pieces,omitempty"            jsonschema:"The count of pieces of the text."`
}

type versionOutput struct {
	Number int    `json:"number" jsonschema:"The version number, from 0."`
	Date   string `json:"date"   jsonschema:"The publication date of the version, YYYY-MM-DD."`
}

type headingOutput struct {
	Kind   string `json:"kind"   jsonschema:"section, chapter, or article."`
	Number string `json:"number" jsonschema:"The number, for example IV or 133^3."`
	Title  string `json:"title"  jsonschema:"The title of the part."`
}

type articleOutput struct {
	Number     string   `json:"number"     jsonschema:"The article number, for example 133^3."`
	Title      string   `json:"title"      jsonschema:"The title of the article."`
	Paragraphs []string `json:"paragraphs" jsonschema:"The paragraphs of the article as published, with the notes of each amendment."`
}
