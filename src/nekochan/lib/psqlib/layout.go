package psqlib

import (
	"fmt"
	"mime"
	gmail "net/mail"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/mail"
	"nekochan/lib/mailib"
)

func attachmentDisposition(original string) string {
	return mime.FormatMediaType(
		"inline", map[string]string{"filename": original})
}

const plainUTF8Type = "text/plain; charset=UTF-8"

func (sp *PSQLIB) fillWebPostDetails(
	i mailib.PostInfo, board string, ref CoreMsgIDStr) mailib.PostInfo {

	hastext := len(i.MI.Message) != 0
	text8bit := !au.Is7BitString(i.MI.Message)

	if i.H != nil {
		panic("header should be nil at this point")
	}

	i.H = make(mail.Headers)

	// we don't really need to store Message-ID there

	// we don't really need to store Subject there

	// From
	// XXX should we hardcode "Anonymous" incase Author is empty?
	i.H["From"] = mail.OneHeaderVal((&gmail.Address{
		Name:    i.MI.Author,
		Address: "poster@" + sp.instance,
	}).String())

	// Newsgroups
	i.H["Newsgroups"] = mail.OneHeaderVal(board)

	// Date
	{
		dd := i.Date
		Y, M, D := dd.Date()
		h, m, s := dd.Clock()
		i.H["Date"] = mail.OneHeaderVal(
			fmt.Sprintf(
				"%02d %s %04d %02d:%02d:%02d GMT",
				D, M.String()[:3], Y, h, m, s))
	}

	// References
	if ref != "" {
		i.H["References"] = mail.OneHeaderVal(fmt.Sprintf("<%s>", ref))
	}

	// X-Sage
	if i.MI.Sage && ref != "" {
		// NOTE: some impls specifically check for "1"
		i.H["X-Sage"] = mail.OneHeaderVal("1")
	}

	// Path
	i.H["Path"] = mail.OneHeaderVal(sp.instance + "!.POSTED!not-for-mail")

	// now deal with layout

	if len(i.FI) == 0 {
		if !hastext {
			// empty. don't include Content-Type header either
			i.L.Body.Data = nil
		} else {
			i.L.Body.Data = mailib.PostObjectIndex(0)
			if text8bit {
				i.H["Content-Type"] = mail.OneHeaderVal(plainUTF8Type)
			}
		}
		return i
	}

	// {RFC 2183}
	// 2.10  Content-Disposition and the Main Message
	//   It is permissible to use Content-Disposition on the main body of an
	//   [RFC 822] message.
	//
	// I wonder how well this will work in pratice
	if !hastext && len(i.FI) == 1 {
		// single attachment
		if len(i.FI[0].ContentType) == 0 {
			panic("Content-Type not set")
		}
		i.H["Content-Type"] = mail.OneHeaderVal(i.FI[0].ContentType)
		i.H["Content-Disposition"] =
			mail.OneHeaderVal(attachmentDisposition(i.FI[0].Original))
		i.L.Body.Data = mailib.PostObjectIndex(1)
		i.L.Binary = true
		return i
	}

	nparts := len(i.FI)
	if hastext {
		nparts++
	}
	xparts := make([]mailib.PartInfo, nparts)
	x := 0
	if hastext {
		if text8bit {
			xparts[0].ContentType = plainUTF8Type
		}
		xparts[0].Body.Data = mailib.PostObjectIndex(0)
		x++
	}
	for a := range i.FI {
		if len(i.FI[a].ContentType) == 0 {
			panic("Content-Type not set")
		}
		xparts[x].ContentType = i.FI[x].ContentType
		xparts[x].Headers = mail.Headers{
			"Content-Disposition": mail.OneHeaderVal(
				attachmentDisposition(i.FI[x].Original)),
		}
		xparts[x].Body.Data = mailib.PostObjectIndex(1 + a)
		xparts[x].Binary = true
		x++
	}
	i.H["Content-Type"] = mail.OneHeaderVal("multipart/mixed")
	i.L.Body.Data = xparts
	return i
}
