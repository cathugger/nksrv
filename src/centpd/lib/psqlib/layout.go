package psqlib

import (
	"fmt"
	"mime"

	au "centpd/lib/asciiutils"
	"centpd/lib/mail"
	"centpd/lib/mailib"
)

func attachmentDisposition(oname string) (cdis string) {
	cdis = mime.FormatMediaType(
		"attachment", map[string]string{"filename": oname})
	// if failed to encode, or has invalid chars
	if cdis == "" || au.ContainsControlString(cdis) {
		// escape using mime hackery
		// because apparently mime.FormatMediaType was too shit to do that
		oname = mime.BEncoding.Encode("UTF-8", oname)
		cdis = mime.FormatMediaType(
			"attachment", map[string]string{"filename": oname})
	}
	return
}

func attachmentConentType(ctype string, oname string) string {
	ct, cpar, err := mime.ParseMediaType(ctype)
	if err != nil {
		// cannot parse media type -- cannot add "name" parameter
		return ctype
	}

	cpar["name"] = oname
	ct = mime.FormatMediaType(ct, cpar)

	// if failed to encode, or has invalid chars
	if ct == "" || au.ContainsControlString(ct) {
		// escape using mime hackery
		// because apparently mime.FormatMediaType was too shit to do that
		cpar["name"] = mime.BEncoding.Encode("UTF-8", oname)
		ct = mime.FormatMediaType(ct, cpar)
	}

	if ct != "" {
		return ct
	} else {
		// if formatting failed, return original
		return ctype
	}
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
	i.H["From"] = mail.OneHeaderVal(
		mail.FormatAddress(i.MI.Author, "poster@"+sp.instance))

	// Newsgroups
	i.H["Newsgroups"] = mail.OneHeaderVal(board)

	// Date
	i.H["Date"] = mail.OneHeaderVal(mail.FormatDate(i.Date))

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
				i.L.Has8Bit = true
				i.H["MIME-Version"] = mail.OneHeaderVal("1.0")
				i.H["Content-Type"] = mail.OneHeaderVal(plainUTF8Type)
			}
		}
		return i
	}

	i.H["MIME-Version"] = mail.OneHeaderVal("1.0")

	// {RFC 2183}
	// 2.10  Content-Disposition and the Main Message
	//   It is permissible to use Content-Disposition on the main body of an
	//   [RFC 822] message.
	//
	// however other impls don't really expect that so don't do it
	/*
		if !hastext && len(i.FI) == 1 {
			// single attachment
			if len(i.FI[0].ContentType) == 0 {
				panic("Content-Type not set")
			}
			i.H["Content-Type"] = mail.OneHeaderVal(
				attachmentConentType(i.FI[0].ContentType, i.FI[0].Original))
			i.H["Content-Disposition"] =
				mail.OneHeaderVal(attachmentDisposition(i.FI[0].Original))
			i.L.Body.Data = mailib.PostObjectIndex(1)
			i.L.Binary = true
			return i
		}
	*/

	nparts := len(i.FI)
	if hastext {
		nparts++
	}
	xparts := make([]mailib.PartInfo, nparts)
	x := 0
	if hastext {
		if text8bit {
			xparts[0].Has8Bit = true
			xparts[0].ContentType = plainUTF8Type
		}
		xparts[0].Body.Data = mailib.PostObjectIndex(0)
		x++
	}
	for a := range i.FI {
		if len(i.FI[a].ContentType) == 0 {
			panic("Content-Type not set")
		}
		xparts[x].ContentType = attachmentConentType(
			i.FI[a].ContentType, i.FI[a].Original)
		xparts[x].Headers = mail.Headers{
			"Content-Disposition": mail.OneHeaderVal(
				attachmentDisposition(i.FI[a].Original)),
		}
		xparts[x].Body.Data = mailib.PostObjectIndex(1 + a)
		xparts[x].Binary = true
		x++
	}
	i.H["Content-Type"] = mail.OneHeaderVal("multipart/mixed")
	i.L.Body.Data = xparts
	i.L.Has8Bit = text8bit
	return i
}
