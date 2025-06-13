package bareprints

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

var files = map[string]string{
	"example/example.go": `package example
import "fmt"
func _() {
	println("") // want "found print to stdout: .+"
	print("") // want "found print to stdout: .+"
	fmt.Fprintln(nil)
	fmt.Println("") // want "found print to stdout: .+"
	fmt.Print("") // want "found print to stdout: .+"
	fmt.Printf("") // want "found print to stdout: .+"
}
`,
	"example/excluded_test.go": `package example
import "fmt"
func Example() {
	fmt.Println("Example functions are exempted")
}
func helper(){
	fmt.Println("other functions are not exempted") // want "found print to stdout: .+"
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
