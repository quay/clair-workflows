package findurls

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

var files = map[string]string{
	"example/annotated.go": `package example
//doc:anotation
var annotated = "https://example.com"
//doc:anotation
const annotated_ = "https://example.com"
`,
	"example/noturl.go": `package example
var notaUrl = "asdf"
const notaUrl_ = "asdf"
`,
	"example/quote.go": `package example
var notAnnotated = "https://example.com" // want "found URL-like string without annotation: .+"
const notAnnotated_ = "https://example.com" // want "found URL-like string without annotation: .+"
`,
	"example/backtick.go": "package example\n" +
		"var notAnnotatedBacktick = `https://example.com` // want `found URL-like string without annotation: .+`\n" +
		"const notAnnotatedBacktick_ = `https://example.com` // want `found URL-like string without annotation: .+`\n",
	"example/in_func.go": `package example
func _() {
	var _ = "https://example.com"
}
`,
}

func TestAnalyzer(t *testing.T) {
	dir, done, err := analysistest.WriteFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(done)
	analysistest.Run(t, dir, Analyzer, "example")
}
