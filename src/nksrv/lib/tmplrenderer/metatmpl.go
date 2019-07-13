package tmplrenderer

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"

	"golang.org/x/crypto/blake2b"
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
	env         *NodeInfo
	staticdir   string
}

func metaTmplAndFile(
	dir, name string) (*template.Template, string) {

	f, err := ioutil.ReadFile(path.Join(dir, name+".tmpl"))
	if err != nil {
		panic("ioutil.ReadFile fail: " + err.Error())
	}

	return template.New(name).Delims("{%", "%}"), string(f)
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

	t, f := metaTmplAndFile(mc.dir, "captcha-"+mc.captchamode)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapMCEnv(mc),
		"static":  wrapMCStatic(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(template.Must(t.Parse(f)), x)
}

func wrapMCInvokeCaptcha(
	mc metaContext) func(x ...interface{}) (string, error) {

	return func(x ...interface{}) (string, error) {
		var xx interface{}
		if len(x) != 0 {
			xx = x[0]
		}
		return invokeCaptcha(mc, xx)
	}
}

func invokePart(
	mc metaContext, name string, x interface{}) (string, error) {

	t, f := metaTmplAndFile(mc.dir, "part-"+name)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapMCEnv(mc),
		"static":  wrapMCStatic(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(template.Must(t.Parse(f)), x)
}

func wrapMCInvokePart(
	mc metaContext) func(name string, x ...interface{}) (string, error) {

	return func(name string, x ...interface{}) (string, error) {
		var xx interface{}
		if len(x) != 0 {
			xx = x[0]
		}
		return invokePart(mc, name, xx)
	}
}

func wrapMCEnv(mc metaContext) func() interface{} {
	return func() interface{} { return mc.env }
}

func hashStaticFile(mc metaContext, name string) (string, error) {
	fname := mc.staticdir + name
	f, e := os.Open(fname)
	if e != nil {
		return "", fmt.Errorf("error opening %q: %v", fname, e)
	}

	const hashlen = 8

	b, e := blake2b.New(hashlen, nil)
	if e != nil {
		panic(e)
	}

	_, e = io.Copy(b, f)
	if e != nil {
		return "", fmt.Errorf("error reading %q: %v", fname, e)
	}

	var sum [hashlen]byte
	b.Sum(sum[:0])

	return mc.env.Root + "/_static/" + name + "?v=" + hex.EncodeToString(sum[:]), nil
}

func wrapMCStatic(mc metaContext) func(string) (string, error) {
	return func(name string) (string, error) {
		return hashStaticFile(mc, name)
	}
}

func invokePage(mc metaContext, name, part string) (string, error) {
	if part != "" {
		name = "page-" + name + "-" + part
	} else {
		name = "page-" + name
	}
	t, f := metaTmplAndFile(mc.dir, name)

	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapMCEnv(mc),
		"static":  wrapMCStatic(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
	})
	return execToString(template.Must(t.Parse(f)), mc.env)
}

func invokeBase(mc metaContext, base, name string) (string, error) {
	t, f := metaTmplAndFile(mc.dir, "base-"+base)
	t.Funcs(template.FuncMap{
		"list": f_list,
		"dict": f_dict,
		"map":  f_dict,

		"env":     wrapMCEnv(mc),
		"static":  wrapMCStatic(mc),
		"invoke":  wrapMCInvokePart(mc),
		"captcha": wrapMCInvokeCaptcha(mc),
		"page": func(part string) (string, error) {
			return invokePage(mc, name, part)
		},
	})
	return execToString(template.Must(t.Parse(f)), mc.env)
}

func loadMetaTmpl(
	mc metaContext, base, name string) (string, error) {

	if base != "" {
		return invokeBase(mc, base, name)
	} else {
		return invokePage(mc, name, "")
	}
}
