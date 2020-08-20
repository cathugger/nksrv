package pibaseweb

import (
	"nksrv/lib/mail/form"
	"nksrv/lib/webcaptcha"
	ib0 "nksrv/lib/webib0"
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
