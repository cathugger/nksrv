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

type metaContext struct {
	dir         string
	captchamode string
}

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

func invokeCaptcha(mc metaContext, x interface{}) (string, error) {
	if mc.captchamode == "" {
		return "", nil
	}

	t := metaTmplFromFile(mc.dir, "captcha_"+mc.captchamode)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

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

func invokePart(mc metaContext, name string, x interface{}) (string, error) {
	t := metaTmplFromFile(mc.dir, "part_"+name)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

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

func invokePage(
	mc metaContext, name, part string, env interface{}) (string, error) {

	if part != "" {
		name = "page_" + name + "_" + part
	} else {
		name = "page_" + name
	}
	t := metaTmplFromFile(mc.dir, name)

	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(t, env)
}

func invokeBase(
	mc metaContext, base, name string, env interface{}) (string, error) {

	t := metaTmplFromFile(mc.dir, "base_"+name)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
		"page": func(part string, env interface{}) (string, error) {
			return invokePage(mc, name, part, env)
		},
	})
	return execToString(t, env)
}

func loadMetaTmpl(
	mc metaContext, base, name string, env interface{}) (string, error) {

	if base != "" {
		return invokeBase(mc, base, name, env)
	} else {
		return invokePage(mc, name, "", env)
	}
}
