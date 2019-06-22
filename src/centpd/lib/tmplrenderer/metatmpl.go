package tmplrenderer

import (
	"io/ioutil"
	"strings"
	"text/template"
)

/*
 * 2 call types:
 * + one invoking page part; automatically picks environment as arg
 * + one invoking include; argument is manually passed
 *
 * base-templates `base_$name`:
 * - is given "environment"
 * - can invoke "pages"
 * - stuff invoked either inherit env
 * page templates `page_$name[_$sub]`
 * - is given environment
 * - can invoke "parts" passing stuff to it
 *
 * im not realy sure if distinction should even be there
 * but separating these could give some saner ordering methinks
 */

func metaTmplFromFile(dir, name string) *template.Template {
	f, err := ioutil.ReadFile(dir + name + ".tmpl")
	if err != nil {
		panic("ioutil.ReadFile fail: " + err.Error())
	}

	return template.Must(
		template.New(name).
		Delims("{%", "%}").
		Parse(string(f)))
}

func execToString(t *template.Template, x interface{}) (string, error) {
	var b strings.Builder
	err := t.Execute(&b, x)
	return b.String(), err
}

func invokePart(dir, name string, x interface{}) (string, error) {
	t := metaTmplFromFile(dir, "part_" + name)
	t.Funcs(template.FuncMap{
		"invoke": wrapDirInvokePart(dir),
	})
	return execToString(t, x)
}

func wrapDirInvokePart(
	dir string) func(name string, x interface{}) (string, error) {

	return func(name string, x interface{}) (string, error) {
		return invokePart(dir, name, x)
	}
}

func invokePage(
	dir, name, part string, env interface{}) (string, error) {

	if part != "" {
		name = "page_" + name + "_" + part
	} else {
		name = "page_" + name
	}
	t := metaTmplFromFile(dir, name)

	t.Funcs(template.FuncMap{
		"invoke": wrapDirInvokePart(dir),
	})
	return execToString(t, env)
}

func parseBase(dir, base, name string, env interface{}) (string, error) {
	t := metaTmplFromFile(dir, "base_" + name)
	t.Funcs(template.FuncMap{
		"invoke": wrapDirInvokePart(dir),
		"page": func(part string, env interface{}) (string, error) {
			return invokePage(dir, name, part, env)
		},
	})
	return execToString(t, env)
}

func loadMetaTmpl(
	dir, base, name string, env interface{}) (string, error) {

	if base != "" {
		return parseBase(dir, base, name, env)
	} else {
		return invokePage(dir, name, "", env)
	}
}
