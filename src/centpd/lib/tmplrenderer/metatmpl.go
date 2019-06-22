package tmplrenderer

import (
	"io/ioutil"
	"path"
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

type metaContext struct {
	dir         string
	captchamode string
	env         interface{}
}

func metaTmplFromFile(dir, name string) *template.Template {
	f, err := ioutil.ReadFile(path.Join(dir, name+".tmpl"))
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

func invokeCaptcha(mc metaContext, x interface{}) (string, error) {
	if mc.captchamode == "" {
		return "", nil
	}

	t := metaTmplFromFile(mc.dir, "captcha-"+mc.captchamode)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapEnv(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(t, x)
}

func wrapMCInvokeCaptcha(
	mc metaContext) func(x interface{}) (string, error) {

	return func(x interface{}) (string, error) {
		return invokeCaptcha(mc, x)
	}
}

func invokePart(
	mc metaContext, name string, x interface{}) (string, error) {

	t := metaTmplFromFile(mc.dir, "part-"+name)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapEnv(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(t, x)
}

func wrapMCInvokePart(
	mc metaContext) func(name string, x interface{}) (string, error) {

	return func(name string, x interface{}) (string, error) {
		return invokePart(mc, name, x)
	}
}

func wrapEnv(mc metaContext) func() interface{} {
	return func() interface{} { return mc.env }
}

func invokePage(
	mc metaContext, name, part string) (string, error) {

	if part != "" {
		name = "page-" + name + "-" + part
	} else {
		name = "page-" + name
	}
	t := metaTmplFromFile(mc.dir, name)

	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapEnv(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(t, mc.env)
}

func invokeBase(mc metaContext, base, name string) (string, error) {

	t := metaTmplFromFile(mc.dir, "base-"+name)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapEnv(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
		"page": func(part string) (string, error) {
			return invokePage(mc, name, part)
		},
	})
	return execToString(t, mc.env)
}

func loadMetaTmpl(
	mc metaContext, base, name string) (string, error) {

	if base != "" {
		return invokeBase(mc, base, name)
	} else {
		return invokePage(mc, name, "")
	}
}
