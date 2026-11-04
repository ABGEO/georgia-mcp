package tool

type searchTablesInput struct {
	Query    string `json:"query"              jsonschema:"The words to find in the title or the folder of a table, in the language of the request. 1 to 100 characters."`
	Language string `json:"language,omitempty" jsonschema:"The language of the titles: ka or en. The default is en."`
	Limit    int    `json:"limit,omitempty"    jsonschema:"The maximum count of tables, 1 to 100. The default is 20."`
}

type tablesOutput struct {
	Source string        `json:"source" jsonschema:"The source of the data: Geostat."`
	AsOf   string        `json:"as_of"  jsonschema:"The instant at which the server read the catalog, RFC 3339."`
	Stale  bool          `json:"stale"  jsonschema:"True when Geostat did not answer the last read of the catalog."`
	Total  int           `json:"total"  jsonschema:"The count of tables that match. More than the count of tables when the limit cut the list."`
	Tables []tableOutput `json:"tables" jsonschema:"The matching tables, the newest update first."`
}

type getTableInput struct {
	Code     string `json:"code"               jsonschema:"The code of a table, from stats_search_tables: Social Statistics/Labour/Wages_Region.px."`
	Language string `json:"language,omitempty" jsonschema:"The language of the texts: ka or en. The default is en."`
}

type tableOutput struct {
	Source     string            `json:"source,omitempty"     jsonschema:"The source of the data: Geostat."`
	AsOf       string            `json:"as_of,omitempty"      jsonschema:"The instant at which the server read the table, RFC 3339."`
	Stale      bool              `json:"stale,omitempty"      jsonschema:"True when Geostat did not answer and the server returns a table it read before."`
	Code       string            `json:"code"                 jsonschema:"The code of the table."`
	Title      string            `json:"title"                jsonschema:"The title of the table. It often names the unit."`
	Folder     string            `json:"folder"               jsonschema:"The folder path of the table, from the subject down, as in the code. The same in both languages."`
	Updated    string            `json:"updated"              jsonschema:"The date on which Geostat last changed the table, YYYY-MM-DD."`
	Dimensions []dimensionOutput `json:"dimensions,omitempty" jsonschema:"The dimensions of the table, each with its values in the order of Geostat."`
}

type dimensionOutput struct {
	Code   string        `json:"code"             jsonschema:"The code of the dimension, for a selection."`
	Text   string        `json:"text"             jsonschema:"The name of the dimension."`
	Values []valueOutput `json:"values,omitempty" jsonschema:"The values of the dimension."`
}

type valueOutput struct {
	Code string `json:"code" jsonschema:"The code of the value, for a selection."`
	Text string `json:"text" jsonschema:"The text of the value."`
}

type getCellsInput struct {
	Code      string           `json:"code"                jsonschema:"The code of a table, from stats_search_tables."`
	Selection []selectionInput `json:"selection,omitempty" jsonschema:"The values of each dimension. A dimension that this list omits takes every value."`
	Language  string           `json:"language,omitempty"  jsonschema:"The language of the texts: ka or en. The default is en."`
}

type selectionInput struct {
	Dimension string   `json:"dimension"        jsonschema:"The code of a dimension, from stats_get_table."`
	Values    []string `json:"values,omitempty" jsonschema:"The codes of the values. Give values or last, not both."`
	Last      int      `json:"last,omitempty"   jsonschema:"The count of the last values of the dimension, the newest for a time dimension."`
}

type cellsOutput struct {
	Source     string            `json:"source"     jsonschema:"The source of the data: Geostat."`
	AsOf       string            `json:"as_of"      jsonschema:"The instant at which the server read the figures, RFC 3339."`
	Stale      bool              `json:"stale"      jsonschema:"True when Geostat did not answer and the server returns figures it read before."`
	Table      tableOutput       `json:"table"      jsonschema:"The table, with no dimension values."`
	Dimensions []dimensionOutput `json:"dimensions" jsonschema:"The dimensions of each cell, in the order of the values of a cell."`
	Cells      []cellOutput      `json:"cells"      jsonschema:"The selected cells."`
}

type cellOutput struct {
	Values []string `json:"values" jsonschema:"The text of the value of the cell on each dimension, in the order of dimensions."`
	Figure *string  `json:"figure" jsonschema:"The figure as a decimal string, or null when Geostat did not publish it."`
}

type getInflationInput struct {
	Month    string `json:"month,omitempty"    jsonschema:"The month, YYYY-MM. The default is the newest published month."`
	Region   string `json:"region,omitempty"   jsonschema:"georgia, tbilisi, kutaisi, batumi, gori, telavi, or zugdidi. The default is georgia."`
	Group    string `json:"group,omitempty"    jsonschema:"The price group: total, food, alcohol_tobacco, clothing, housing, furnishings, health, transport, communication, recreation, education, restaurants, or miscellaneous. The default is total."`
	Language string `json:"language,omitempty" jsonschema:"The language of the names: ka or en. The default is en."`
}

type inflationOutput struct {
	Source  string        `json:"source"  jsonschema:"The source of the data: Geostat."`
	AsOf    string        `json:"as_of"   jsonschema:"The instant at which the server read the indices, RFC 3339."`
	Stale   bool          `json:"stale"   jsonschema:"True when Geostat did not answer and the server returns indices it read before."`
	Month   string        `json:"month"   jsonschema:"The month, YYYY-MM."`
	Region  namedOutput   `json:"region"  jsonschema:"The region."`
	Group   namedOutput   `json:"group"   jsonschema:"The price group."`
	Annual  *string       `json:"annual"  jsonschema:"The change of prices since the same month of the previous year, in percent. Null when not published."`
	Monthly *string       `json:"monthly" jsonschema:"The change of prices since the previous month, in percent. Null when not published."`
	Average *string       `json:"average" jsonschema:"The change of the average prices of the last 12 months over the 12 months before, in percent. Published for the group total only."`
	Core    *string       `json:"core"    jsonschema:"The core inflation, in percent. Published for georgia and the group total only."`
	Tables  []tableOutput `json:"tables"  jsonschema:"The tables of Geostat that the answer reads, with their update dates."`
}

type namedOutput struct {
	Code string `json:"code" jsonschema:"The code."`
	Name string `json:"name" jsonschema:"The name in the language of the request."`
}
