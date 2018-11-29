package demousrdb

import "errors"

type DemoUsrDB struct{}

type demoUsr struct {
	pass  string
	attrs map[string]interface{}
}

var demoUsers = map[string]demoUsr{
	"tester": demoUsr{
		pass: "test",
		attrs: map[string]interface{}{
			"admin": false,
		},
	},
	"admin": demoUsr{
		pass: "hackme",
		attrs: map[string]interface{}{
			"admin": true,
		},
	},
}

func (DemoUsrDB) UsrLogin(usr, pass string) (attrs map[string]interface{}, err error) {
	u, ok := demoUsers[usr]
	if !ok || u.pass != pass {
		err = errors.New("invalid login")
		return
	}
	attrs = u.attrs
	attrs["user"] = usr
	return
}
