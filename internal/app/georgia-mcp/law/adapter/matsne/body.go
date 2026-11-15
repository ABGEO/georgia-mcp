package matsne

import (
	"regexp"
	"strings"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/law/domain"
	matsneclient "github.com/abgeo/georgia-mcp/pkg/matsne"
)

const (
	classSection      = "karixml"
	classSectionTitle = "karisataurixml"
	classChapter      = "tavixml"
	classChapterTitle = "tavisataurixml"
	classArticle      = "muxlixml"

	// invisible holds the characters of the page that show nothing: the zero width space,
	// the zero width joiners, the word joiner, the byte order mark, and the soft hyphen.
	invisible = "\u200b\u200c\u200d\u2060\ufeff\u00ad"
)

var (
	georgianSection = regexp.MustCompile(`^კარი\s+([^\s.]+)\.?\s*(.*)$`)
	georgianChapter = regexp.MustCompile(`^თავი\s+([^\s.]+)\.?\s*(.*)$`)
	georgianArticle = regexp.MustCompile(`^მუხლი\s+([^\s.]+)\.?\s*(.*)$`)
	englishSection  = regexp.MustCompile(`^SECTION\s+([^\s–—-]+)\s*(?:[–—-]\s*(.*))?$`)
	englishChapter  = regexp.MustCompile(`^Chapter\s+(\S+)\s*[–—-]\s*(.*)$`)
	englishArticle  = regexp.MustCompile(`^Article\s+(\S+)\s*[–—-]\s*(.*)$`)
)

// bodyBuilder collects the headings and the articles in the order of the page. Each
// paragraph after an article heading belongs to that article, up to the next heading.
type bodyBuilder struct {
	body domain.Body
	// all holds every paragraph, for an act with no article.
	all []string
	// open is the index of the open article, or -1.
	open int
	// titled is the index of the heading that waits for its title, or -1.
	titled int
}

func newBodyBuilder() *bodyBuilder {
	return &bodyBuilder{open: -1, titled: -1}
}

func (b *bodyBuilder) part(kind domain.PartKind, number, title string) {
	b.body.Contents = append(
		b.body.Contents,
		domain.Heading{Kind: kind, Number: number, Title: title},
	)
	b.open = -1
	b.titled = -1

	if title == "" {
		b.titled = len(b.body.Contents) - 1
	}
}

func (b *bodyBuilder) title(text string) bool {
	if b.titled < 0 {
		return false
	}

	b.body.Contents[b.titled].Title = text
	b.titled = -1

	return true
}

// article opens an article. A heading whose number does not parse stays a paragraph.
func (b *bodyBuilder) article(number, title, text string) {
	parsed, err := domain.ParseArticleNumber(number)
	if err != nil {
		b.paragraph(text)

		return
	}

	b.body.Contents = append(b.body.Contents, domain.Heading{
		Kind: domain.KindArticle, Number: parsed.String(), Title: title,
	})
	b.body.Articles = append(b.body.Articles, domain.Article{Number: parsed, Title: title})
	b.open = len(b.body.Articles) - 1
	b.titled = -1
}

func (b *bodyBuilder) paragraph(text string) {
	b.titled = -1

	if b.open >= 0 {
		article := &b.body.Articles[b.open]
		article.Paragraphs = append(article.Paragraphs, text)
	}
}

// build returns the body. An act with no article gives its text alone.
func (b *bodyBuilder) build() domain.Body {
	if len(b.body.Articles) == 0 {
		return domain.Body{Text: strings.Join(b.all, "\n")}
	}

	return b.body
}

func georgianBody(paragraphs []matsneclient.Paragraph) domain.Body {
	builder := newBodyBuilder()

	for _, paragraph := range paragraphs {
		text := clean(paragraph.Text)
		if text == "" {
			continue
		}

		builder.all = append(builder.all, text)

		switch paragraph.Class {
		case classSection:
			number, title := heading(georgianSection, text)
			builder.part(domain.KindSection, number, title)
		case classChapter:
			number, title := heading(georgianChapter, text)
			builder.part(domain.KindChapter, number, title)
		case classSectionTitle, classChapterTitle:
			georgianTitle(builder, paragraph.Class, text)
		case classArticle:
			if match := georgianArticle.FindStringSubmatch(text); match != nil {
				builder.article(match[1], match[2], text)
			} else {
				builder.paragraph(text)
			}
		default:
			builder.paragraph(text)
		}
	}

	return builder.build()
}

// englishBody reads the Word markup of a translation. Only a bold paragraph is a heading,
// so a citation such as "under Article 281 of this Code" opens no article.
func englishBody(paragraphs []matsneclient.Paragraph) domain.Body {
	builder := newBodyBuilder()

	for _, paragraph := range paragraphs {
		text := clean(paragraph.Text)
		if text == "" {
			continue
		}

		builder.all = append(builder.all, text)

		if builder.title(text) {
			continue
		}

		if !paragraph.Bold {
			builder.paragraph(text)

			continue
		}

		if match := englishSection.FindStringSubmatch(text); match != nil {
			builder.part(domain.KindSection, match[1], match[2])

			continue
		}

		if match := englishChapter.FindStringSubmatch(text); match != nil {
			builder.part(domain.KindChapter, match[1], match[2])
		} else if match := englishArticle.FindStringSubmatch(text); match != nil {
			builder.article(match[1], match[2], text)
		} else {
			builder.paragraph(text)
		}
	}

	return builder.build()
}

// heading returns the number and the title of "კარი I" or "კარი I title". A text that does
// not match gives the whole text as the number.
func heading(pattern *regexp.Regexp, text string) (string, string) {
	match := pattern.FindStringSubmatch(text)
	if match == nil {
		return text, ""
	}

	return match[1], match[2]
}

// georgianTitle gives the title to the heading above it. A page can leave the heading
// empty and put "თავი X. title" in the title paragraph, which then opens the part itself.
func georgianTitle(builder *bodyBuilder, class, text string) {
	if builder.title(text) {
		return
	}

	kind, pattern := domain.KindChapter, georgianChapter
	if class == classSectionTitle {
		kind, pattern = domain.KindSection, georgianSection
	}

	if match := pattern.FindStringSubmatch(text); match != nil {
		builder.part(kind, match[1], match[2])

		return
	}

	builder.paragraph(text)
}

// clean removes the invisible characters and folds each run of white space to one space.
func clean(text string) string {
	visible := strings.Map(func(char rune) rune {
		if strings.ContainsRune(invisible, char) {
			return -1
		}

		return char
	}, text)

	return strings.Join(strings.Fields(visible), " ")
}
