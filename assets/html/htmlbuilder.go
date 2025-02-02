package html

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

// If this is your first time seeing
// the embed package take a look at the official docs:
//
// - https://pkg.go.dev/embed
//
// In short the go embed package will load files during
// compilation and save them inside the binary so you
// deploy it to production without having to find a way of
// copying the HTML there too.
//go:embed *
var htmlFS embed.FS

var exampleTemplate = mustParseTemplate("example.html")

// WriteExamplePage
func WriteExamplePage(
	w io.Writer,
	v1 string,
	v2 string,
	v3 int,
) error {
	return exampleTemplate.Execute(w, map[string]interface{}{
		"var1": v1,
		"var2": v2,
		"var3": v3,
	})
}

// mustParseTemplate is used to simplify the loading of all
// required files, if a file is not present or if it is not
// well formatted it will panic.
func mustParseTemplate(filename string) *template.Template {
	data, err := htmlFS.ReadFile(filename)
	if err != nil {
		panic(
			fmt.Sprintf("unable to find HTML template '%s': %s", filename, err),
		)
	}

	t, err := template.New(filename).Parse(string(data))
	if err != nil {
		panic(
			fmt.Sprintf("unable to parse HTML template '%s': %s", filename, err),
		)
	}

	return t
}
