package codegen

import (
	"embed"
	"fmt"
	"strconv"
	"text/template"
)

//go:embed _templates/*.go.tmpl
var embeddedTemplatesFS embed.FS

// Template returns the embedded template used for code generation.
func Template() *template.Template {
	t := template.New("main.go.tmpl").Option("missingkey=error").Funcs(template.FuncMap{
		"backquote":   backquote,
		"quote":       strconv.Quote,
		"quoteSingle": quoteSingle,
	})
	return template.Must(t.ParseFS(embeddedTemplatesFS, "_templates/*"))
}

// backquote returns backquoted string if possible, otherwise it returns an
// error.
func backquote(s string) (string, error) {
	if !strconv.CanBackquote(s) {
		return "", fmt.Errorf("cannot backquote string %q", s)
	}
	return "`" + s + "`", nil
}

// quoteSingle is like [strconv.Quote], but uses single quotes instead of double
// quotes.
func quoteSingle(s string) string {
	const escape, single, double = '\\', '\'', '"'

	buf := strconv.AppendQuote(nil, s)
	end := len(buf) - 1

	delta, changed := 0, false
	for i := end - 1; i > 0; i-- {
		switch {
		case buf[i] == single:
			changed = true
			delta++
		case buf[i-1] == escape && buf[i] == double:
			changed = true
			delta--
		}
	}

	if !changed {
		buf[0], buf[len(buf)-1] = single, single
		return string(buf)
	}

	if delta > 0 {
		buf = append(buf, make([]byte, delta)...)
	}

	n := len(buf) - 1
	buf[n] = single
	n--

	for i := end - 1; i > 0; i-- {
		buf[n] = buf[i]
		n--
		switch {
		case buf[i] == single:
			buf[n] = escape
			n-- // insert escape
		case buf[i-1] == escape && buf[i] == double:
			i-- // remove escape
		}
	}
	buf[n] = single

	return string(buf[n:])
}
