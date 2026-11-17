package gnerc

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// maxPart bounds each part of a workbook after it expands.
const maxPart = 16 << 20

const (
	workbookPart  = "xl/workbook.xml"
	relationsPart = "xl/_rels/workbook.xml.rels"
	stringsPart   = "xl/sharedStrings.xml"
	sharedString  = "s"
	inlineString  = "inlineStr"
)

type workbookXML struct {
	Sheets []struct {
		Name     string `xml:"name,attr"`
		Relation string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	} `xml:"sheets>sheet"`
}

type relationsXML struct {
	Relations []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

// richText is a shared string or an inline string: one text, or runs of text. A run of
// phonetic text (rPh) is not part of the string.
type richText struct {
	Text string `xml:"t"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
}

func (r richText) String() string {
	var text strings.Builder

	text.WriteString(r.Text)

	for _, run := range r.Runs {
		text.WriteString(run.Text)
	}

	return text.String()
}

type stringsXML struct {
	Items []richText `xml:"si"`
}

type sheetXML struct {
	Rows []struct {
		Number int       `xml:"r,attr"`
		Cells  []cellXML `xml:"c"`
	} `xml:"sheetData>row"`
}

func readWorkbook(body []byte) ([]Sheet, error) {
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotWorkbook, err)
	}

	var (
		book      workbookXML
		relations relationsXML
		shared    stringsXML
	)

	for name, target := range map[string]any{
		workbookPart: &book, relationsPart: &relations, stringsPart: &shared,
	} {
		if err := decodePart(archive, name, target); err != nil {
			return nil, err
		}
	}

	targets := make(map[string]string, len(relations.Relations))
	for _, relation := range relations.Relations {
		targets[relation.ID] = partOf(relation.Target)
	}

	sheets := make([]Sheet, 0, len(book.Sheets))

	for _, entry := range book.Sheets {
		target, ok := targets[entry.Relation]
		if !ok {
			return nil, fmt.Errorf("%w: the sheet %q has no part", ErrNotWorkbook, entry.Name)
		}

		var data sheetXML
		if err := decodePart(archive, target, &data); err != nil {
			return nil, err
		}

		sheet, err := sheetOf(entry.Name, data, shared.Items)
		if err != nil {
			return nil, err
		}

		sheets = append(sheets, sheet)
	}

	return sheets, nil
}

// partOf returns the name of a part from the target of a relation of the workbook, which
// is relative to the folder xl, or absolute from the root of the archive.
func partOf(target string) string {
	if absolute, ok := strings.CutPrefix(target, "/"); ok {
		return absolute
	}

	return path.Join("xl", target)
}

func decodePart(archive *zip.Reader, name string, target any) error {
	file, err := archive.Open(name)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotWorkbook, err)
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxPart+1))
	if err != nil {
		return fmt.Errorf("%w: reading %s: %w", ErrNotWorkbook, name, err)
	}

	if len(data) > maxPart {
		return fmt.Errorf("%w: %s holds more than %d bytes", ErrNotWorkbook, name, maxPart)
	}

	if err := xml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%w: decoding %s: %w", ErrNotWorkbook, name, err)
	}

	return nil
}

type cellXML struct {
	Reference string    `xml:"r,attr"`
	Type      string    `xml:"t,attr"`
	Value     *string   `xml:"v"`
	Inline    *richText `xml:"is"`
}

func sheetOf(name string, data sheetXML, shared []richText) (Sheet, error) {
	sheet := Sheet{Name: name}
	previous := 0

	for _, source := range data.Rows {
		number := source.Number
		if number == 0 {
			number = previous + 1
		}

		previous = number
		row := Row{Number: number}

		for _, cell := range source.Cells {
			text, ok, err := textOf(cell, shared)
			if err != nil {
				return Sheet{}, fmt.Errorf("the cell %s of %q: %w", cell.Reference, name, err)
			}

			if ok {
				row.Cells = append(row.Cells, Cell{
					Column: strings.TrimRight(cell.Reference, "0123456789"),
					Type:   cell.Type,
					Text:   text,
				})
			}
		}

		if len(row.Cells) > 0 {
			sheet.Rows = append(sheet.Rows, row)
		}
	}

	return sheet, nil
}

// textOf returns the text of a cell, and false for a cell with no value.
func textOf(cell cellXML, shared []richText) (string, bool, error) {
	switch {
	case cell.Type == inlineString && cell.Inline != nil:
		return cell.Inline.String(), true, nil
	case cell.Value == nil:
		return "", false, nil
	case cell.Type == sharedString:
		index, err := strconv.Atoi(*cell.Value)
		if err != nil || index < 0 || index >= len(shared) {
			return "", false, fmt.Errorf("%w: no shared string %q", ErrNotWorkbook, *cell.Value)
		}

		return shared[index].String(), true, nil
	default:
		return *cell.Value, true, nil
	}
}
