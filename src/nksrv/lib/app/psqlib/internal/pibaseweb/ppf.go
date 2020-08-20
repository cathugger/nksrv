package pibaseweb

import (
	"nksrv/lib/app/base/webcaptcha"
	ib0 "nksrv/lib/app/webib0"
	"nksrv/lib/mail/form"
)

func MakePostParamFunc(c *webcaptcha.WebCaptcha) func(string) bool {
	tfields := []string{
		ib0.IBWebFormTextTitle,
		ib0.IBWebFormTextName,
		ib0.IBWebFormTextMessage,
		ib0.IBWebFormTextOptions,
	}
	if c != nil {
		tfields = append(tfields, c.TextFields()...)
	}
	return form.FieldsCheckFunc(tfields)
}
