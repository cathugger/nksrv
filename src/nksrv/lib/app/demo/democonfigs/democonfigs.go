package democonfigs

import (
	"nksrv/lib/app/base/altthumber"
	"nksrv/lib/app/base/captchainfo"
	"nksrv/lib/app/demo/demoib"
	"nksrv/lib/app/psqlib/piconfig"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/gothm"
	"nksrv/lib/utils/fs/fstore"
)

// configs commonly used in demo executables

var CfgAltThm = altthumber.AltThumber(demoib.DemoAltThumber{})

var CfgPSQLIB = piconfig.Config{
	NodeName:   "nekochan",
	SrcCfg:     &fstore.Config{"_demo/demoib0/src"},
	ThmCfg:     &fstore.Config{"_demo/demoib0/thm"},
	NNTPFSCfg:  &fstore.Config{"_demo/demoib0/nntp"},
	AltThumber: &CfgAltThm,
	TBuilder:   gothm.DefaultConfig,
	TCfgOP: &thumbnailer.ThumbConfig{
		Width:       250,
		Height:      250,
		AudioWidth:  350,
		AudioHeight: 350,
		Color:       "#EEF2FF",
	},
	TCfgPost: &thumbnailer.ThumbConfig{
		Width:       200,
		Height:      200,
		AudioWidth:  350,
		AudioHeight: 350,
		Color:       "#D6DAF0",
	},
}

var CfgCaptchaInfo = captchainfo.CaptchaInfo{
	Width:  300,
	Height: 95,
}
