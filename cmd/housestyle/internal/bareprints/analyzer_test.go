package bareprints

import (
	"fmt"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// Test function
func _() {
	println("")
	print("")
	fmt.Fprintln(nil)
	fmt.Println("")
	fmt.Print("")
	fmt.Printf("")
}

var files = map[string]string{
	"example/example.go": `package example

import (
	"fmt"
)

// Test function
func _() {
	println("") // want "found print to stdout: .+"
	print("") // want "found print to stdout: .+"
	fmt.Fprintln(nil)
	fmt.Println("") // want "found print to stdout: .+"
	fmt.Print("") // want "found print to stdout: .+"
	fmt.Printf("") // want "found print to stdout: .+"
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
