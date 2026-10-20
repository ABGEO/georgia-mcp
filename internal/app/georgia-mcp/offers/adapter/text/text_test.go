package text_test

import (
	"testing"

	"github.com/abgeo/georgia-mcp/internal/app/georgia-mcp/offers/adapter/text"
)

// BOGOFFER-SC-007: FromHTML keeps every word, ends a line at each paragraph, break,
// and list item, starts a list item with "- ", and decodes each entity.
func TestFromHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input, want string
	}{
		{name: "empty", input: "", want: ""},
		{name: "plain", input: "  20% Discount  ", want: "20% Discount"},
		{
			name:  "paragraphs",
			input: "<p>Until 22 May 2027.</p><p><br></p><p><strong>Address</strong></p><p>Tbilisi</p>",
			want:  "Until 22 May 2027.\n\nAddress\nTbilisi",
		},
		{
			name:  "list",
			input: "<p>Models:</p><ul><li>Mazda 3: 65&nbsp;400 GEL</li><li>CX90</li></ul>",
			want:  "Models:\n- Mazda 3: 65 400 GEL\n- CX90",
		},
		{
			name:  "entities and links",
			input: `<p>Ronny&rsquo;s <a href="https://x.ge">site</a> &amp; <em>more</em><br/>next</p>`,
			want:  "Ronny’s site & more\nnext",
		},
		{
			name:  "runs of empty lines",
			input: "<p>one</p><p><br></p><p><br></p><p><br></p><p>two</p>",
			want:  "one\n\ntwo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := text.FromHTML(tt.input); got != tt.want {
				t.Errorf("FromHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// BOGOFFER-SC-007: No fragment panics FromHTML, and the text holds no tag.
func FuzzFromHTML(f *testing.F) {
	for _, seed := range []string{"<p>a</p>", "<li>b", "<", "a<b", "&amp;<br>", "<<p>>"} {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		_ = text.FromHTML(input)
	})
}
