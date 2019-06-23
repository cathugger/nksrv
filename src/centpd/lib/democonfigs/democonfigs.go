package democonfigs

import (
	"centpd/lib/altthumber"
	"centpd/lib/captchainfo"
	"centpd/lib/demoib"
	"centpd/lib/fstore"
	"centpd/lib/gothumbnailer"
	"centpd/lib/psqlib"
	"centpd/lib/thumbnailer"
)

// configs commonly used in demo executables

var CfgAltThm = altthumber.AltThumber(demoib.DemoAltThumber{})

var CfgPSQLIB = psqlib.Config{
	SrcCfg:     &fstore.Config{"_demo/demoib0/src"},
	ThmCfg:     &fstore.Config{"_demo/demoib0/thm"},
	NNTPFSCfg:  &fstore.Config{"_demo/demoib0/nntp"},
	AltThumber: &CfgAltThm,
	TBuilder:   gothumbnailer.DefaultConfig,
	TCfgThread: &thumbnailer.ThumbConfig{
		Width:  250,
		Height: 250,
		Color:  "#C5EFCF",
	},
	TCfgReply: &thumbnailer.ThumbConfig{
		Width:  200,
		Height: 200,
		Color:  "#DDFFDD",
	},
}

var CfgCaptchaInfo = captchainfo.CaptchaInfo{
	Width:  305,
	Height: 95,
}
