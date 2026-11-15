package tas_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/abgeo/georgia-mcp/pkg/tas"
)

// TASPERM-SC-006: The scanner quotes the keys, reads a date as its milliseconds, keeps the
// escapes, and drops the null character.
func TestToJSON(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{`{a:1,b:"x\"y",c:new Date(5),d:[null,true]}`, `{"a":1,"b":"x\"y","c":5,"d":[null,true]}`},
		{`"a\u0000b"`, `"ab"`},
		{
			`{ name:'java.Error', message:'it\'s "x"' }`,
			`{ "name":"java.Error", "message":"it's \"x\"" }`,
		},
		{`{a:new Date(-5)}`, `{"a":-5}`},
	}

	for _, tt := range tests {
		got, err := tas.ToJSON(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ToJSON(%q) = %q, %v, want %q", tt.in, got, err, tt.want)
		}
	}
}

// TASPERM-SC-006: A broken value is an error.
func TestToJSONFails(t *testing.T) {
	t.Parallel()

	for _, in := range []string{`"open`, `{a b}`, `new Date(x)`, `{a:new Date(5}`, `'open`} {
		if got, err := tas.ToJSON(in); err == nil {
			t.Errorf("ToJSON(%q) = %q, want an error", in, got)
		}
	}
}

// TASPERM-SC-006: The scanner never panics, and valid JSON gives the same value.
func FuzzToJSON(f *testing.F) {
	for _, seed := range []string{`{"a":[1,2.5,"x"]}`, `{a:new Date(5)}`, `"\u0000"`, `{`, `'a'`} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, in string) {
		got, err := tas.ToJSON(in)
		if err != nil || !json.Valid([]byte(in)) || strings.Contains(in, `\u0000`) {
			return
		}

		var want, have any
		if json.Unmarshal([]byte(in), &want) != nil {
			return
		}

		if err := json.Unmarshal([]byte(got), &have); err != nil {
			t.Fatalf("ToJSON(%q) = %q, not JSON: %v", in, got, err)
		}

		if !reflect.DeepEqual(want, have) {
			t.Errorf("ToJSON(%q) = %v, want %v", in, have, want)
		}
	})
}
