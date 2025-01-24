package shquote

import (
	"testing"

	"golang.org/x/text/transform"
)

func Test(t *testing.T) {
	type testcase struct {
		Name string
		In   string
		Want string
	}

	tcs := []testcase{
		{"None", `test`, `test`},
		{"Space", `te st`, `'te st'`},
		{"SpaceWithSingleQuote", `t'e st`, `'t'\''e st'`},
		{"DoubleQuote", `te"st`, `'te"st'`},
		{"Newline", "test\n\n", "'test\n\n'"},
		{"Dollar", `$test`, `'$test'`},
	}

	tf := &Transformer{}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			got, _, err := transform.String(tf, tc.In)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.Want
			t.Logf("got: %#q, want: %#q", got, want)
			if got != want {
				t.Fail()
			}
		})
	}
}
