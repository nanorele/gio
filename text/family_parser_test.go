package text

import (
	"slices"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	type scenario struct {
		variantName string
		input       string
	}
	type testcase struct {
		name      string
		inputs    []scenario
		expected  []string
		shouldErr bool
	}

	for _, tc := range []testcase{
		{
			name: "empty",
			inputs: []scenario{
				{
					variantName: "",
				},
			},
			shouldErr: true,
		},
		{
			name: "comma failure",
			inputs: []scenario{
				{
					variantName: "bare single",
					input:       ",",
				},
				{
					variantName: "bare multiple",
					input:       ",, ,,",
				},
			},
			shouldErr: true,
		},
		{
			name: "comma success",
			inputs: []scenario{
				{
					variantName: "squote",
					input:       "','",
				},
				{
					variantName: "dquote",
					input:       `","`,
				},
			},
			expected: []string{","},
		},
		{
			name: "comma success multiple",
			inputs: []scenario{
				{
					variantName: "squote",
					input:       "',,', ',,'",
				},
				{
					variantName: "dquote",
					input:       `",,", ",,"`,
				},
			},
			expected: []string{",,", ",,"},
		},
		{
			name: "backslashes",
			inputs: []scenario{
				{
					variantName: "bare",
					input:       `\font\\`,
				},
				{
					variantName: "dquote",
					input:       `"\\font\\\\"`,
				},
				{
					variantName: "squote",
					input:       `'\\font\\\\'`,
				},
			},
			expected: []string{`\font\\`},
		},
		{
			name: "invalid backslashes",
			inputs: []scenario{
				{
					variantName: "dquote",
					input:       `"\\""`,
				},
				{
					variantName: "squote",
					input:       `'\\''`,
				},
			},
			shouldErr: true,
		},
		{
			name: "too many quotes",
			inputs: []scenario{
				{
					variantName: "dquote",
					input:       `"""`,
				},
				{
					variantName: "squote",
					input:       `'''`,
				},
			},
			shouldErr: true,
		},
		{
			name: "serif serif's serif\"s",
			inputs: []scenario{
				{
					variantName: "bare",
					input:       `serif, serif's, serif"s`,
				},
				{
					variantName: "squote",
					input:       `'serif', 'serif\'s', 'serif"s'`,
				},
				{
					variantName: "dquote",
					input:       `"serif", "serif's", "serif\"s"`,
				},
			},
			expected: []string{"serif", `serif's`, `serif"s`},
		},
		{
			name: "complex list",
			inputs: []scenario{
				{
					variantName: "bare",
					input:       `Times New Roman, Georgia Common, Helvetica Neue, serif`,
				},
				{
					variantName: "squote",
					input:       `'Times New Roman', 'Georgia Common', 'Helvetica Neue', 'serif'`,
				},
				{
					variantName: "dquote",
					input:       `"Times New Roman", "Georgia Common", "Helvetica Neue", "serif"`,
				},
				{
					variantName: "mixed",
					input:       `Times New Roman, "Georgia Common", 'Helvetica Neue', "serif"`,
				},
				{
					variantName: "mixed with weird spacing",
					input:       `Times New Roman  ,"Georgia Common"              , 'Helvetica Neue' ,"serif"`,
				},
			},
			expected: []string{"Times New Roman", "Georgia Common", "Helvetica Neue", "serif"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p parser
			for _, scen := range tc.inputs {
				t.Run(scen.variantName, func(t *testing.T) {
					actual, err := p.parse(scen.input)
					if (err != nil) != tc.shouldErr {
						t.Errorf("unexpected error state: %v", err)
					}
					if !slices.Equal(tc.expected, actual) {
						t.Errorf("expected\n%q\ngot\n%q", tc.expected, actual)
					}
				})
			}
		})
	}
}

func TestParserEmptyInput(t *testing.T) {
	var p parser
	got, err := p.parse("")
	if err == nil {
		t.Fatalf("expected error for empty input, got %q", got)
	}
	if len(got) != 0 {
		t.Fatalf("expected no faces, got %q", got)
	}
}

func TestParserSingleUnquoted(t *testing.T) {
	var p parser
	got, err := p.parse("Helvetica")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{"Helvetica"}) {
		t.Fatalf("expected [Helvetica], got %q", got)
	}
}

func TestParserSingleQuotedWithComma(t *testing.T) {
	var p parser
	got, err := p.parse(`"Foo, Bar"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{"Foo, Bar"}) {
		t.Fatalf("expected [Foo, Bar], got %q", got)
	}
}

func TestParserQuotedWithEscapedQuote(t *testing.T) {
	var p parser
	got, err := p.parse(`"Fo\"o"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{`Fo"o`}) {
		t.Fatalf("expected [Fo\"o], got %q", got)
	}
}

func TestParserMixedQuotedUnquoted(t *testing.T) {
	var p parser
	got, err := p.parse(`Helvetica, "Times New Roman", 'Courier New', Arial`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"Helvetica", "Times New Roman", "Courier New", "Arial"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParserTrailingComma(t *testing.T) {

	var p parser
	got, err := p.parse("foo,")
	if err == nil {
		t.Fatalf("expected error for trailing comma, got %q", got)
	}
}

func TestParserTrailingCommaWithSpace(t *testing.T) {

	var p parser
	got, err := p.parse("foo, ")
	if err == nil {
		t.Fatalf("expected error for trailing comma+space, got %q", got)
	}
}

func TestParserWhitespaceOnly(t *testing.T) {
	var p parser
	got, err := p.parse("   \t\n   ")
	if err == nil {
		t.Fatalf("expected error for whitespace-only input, got %q", got)
	}
}

func TestParserUnterminatedDoubleQuote(t *testing.T) {
	var p parser
	got, err := p.parse(`"unterminated`)
	if err == nil {
		t.Fatalf("expected error for unterminated dquote, got %q", got)
	}
}

func TestParserUnterminatedSingleQuote(t *testing.T) {
	var p parser
	got, err := p.parse(`'unterminated`)
	if err == nil {
		t.Fatalf("expected error for unterminated squote, got %q", got)
	}
}

func TestParserUnterminatedQuoteAfterEscape(t *testing.T) {
	var p parser
	got, err := p.parse(`"foo\`)
	if err == nil {
		t.Fatalf("expected error for unterminated dquote after backslash, got %q", got)
	}
}

func TestParserVeryLongName(t *testing.T) {
	long := strings.Repeat("a", 10000)
	var p parser
	got, err := p.parse(long)
	if err != nil {
		t.Fatalf("unexpected error for long name: %v", err)
	}
	if !slices.Equal(got, []string{long}) {
		t.Fatalf("expected single long name, got %d entries", len(got))
	}
}

func TestParserVeryLongQuotedName(t *testing.T) {
	long := strings.Repeat("x", 10000)
	var p parser
	got, err := p.parse(`"` + long + `"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{long}) {
		t.Fatalf("expected single long name")
	}
}

func TestParserManyEntries(t *testing.T) {
	var sb strings.Builder
	want := make([]string, 500)
	for i := 0; i < 500; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		name := "f" + strings.Repeat("o", i%10+1)
		want[i] = name
		sb.WriteString(name)
	}
	var p parser
	got, err := p.parse(sb.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("mismatch: got %d, want %d", len(got), len(want))
	}
}

func TestParserBareNameWithInternalSpaces(t *testing.T) {
	var p parser
	got, err := p.parse("Times New Roman")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{"Times New Roman"}) {
		t.Fatalf("expected [Times New Roman], got %q", got)
	}
}

func TestParserBareNameTrimsWhitespace(t *testing.T) {
	var p parser
	got, err := p.parse("   Helvetica   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{"Helvetica"}) {
		t.Fatalf("expected [Helvetica], got %q", got)
	}
}

func TestParserEmptyQuotedString(t *testing.T) {

	var p parser
	got, err := p.parse(`""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{""}) {
		t.Fatalf("expected [\"\"], got %q", got)
	}
}

func TestParserLeadingComma(t *testing.T) {
	var p parser
	got, err := p.parse(",foo")
	if err == nil {
		t.Fatalf("expected error for leading comma, got %q", got)
	}
}

func TestParserDoubleComma(t *testing.T) {
	var p parser
	got, err := p.parse("foo,,bar")
	if err == nil {
		t.Fatalf("expected error for double comma, got %q", got)
	}
}

func TestParserReuseClearsState(t *testing.T) {
	var p parser
	if _, err := p.parse("foo, bar, baz"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := p.parse("alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{"alpha"}) {
		t.Fatalf("parser carried over state: %q", got)
	}
}

func TestParserReuseAfterError(t *testing.T) {
	var p parser
	if _, err := p.parse(`"unterminated`); err == nil {
		t.Fatalf("expected error")
	}
	got, err := p.parse("recovered")
	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}
	if !slices.Equal(got, []string{"recovered"}) {
		t.Fatalf("expected [recovered], got %q", got)
	}
}

func TestParserUnicodeNames(t *testing.T) {
	var p parser
	got, err := p.parse(`"日本語フォント", "한글", Ωmega`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"日本語フォント", "한글", "Ωmega"}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestTokenString(t *testing.T) {
	cases := []struct {
		t    token
		want string
	}{
		{token{kind: tokenStr, value: "foo"}, "foo"},
		{token{kind: tokenComma}, ","},
		{token{kind: tokenEOF}, "EOF"},
		{token{kind: tokenKind(99)}, "unknown"},
	}
	for _, c := range cases {
		if got := c.t.String(); got != c.want {
			t.Errorf("token{%d}.String() = %q, want %q", c.t.kind, got, c.want)
		}
	}
}
