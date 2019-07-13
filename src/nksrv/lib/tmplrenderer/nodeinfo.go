package tmplrenderer

import "nksrv/lib/captchainfo"

type NodeInfo struct {
	Name  string
	Root  string // for web serving, static, and API
	FRoot string // for file serving
	PRoot string // for post submission

	Captcha captchainfo.CaptchaInfo
}
