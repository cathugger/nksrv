package psqlib

import (
	"mime"

	"nekochan/lib/mail"
)

func attachmentDisposition(original string) string {
	return mime.FormatMediaType(
		"inline", map[string]string{"filename": original})
}

const plainType = "text/plain; charset=UTF-8"

func WebPostToLayout(i *postInfo) {
	hastext := len(i.MI.Message) != 0

	if len(i.FI) == 0 {
		if !hastext {
			// empty. don't include Content-Type header either
			i.L.Body.Data = nil
		} else {
			i.L.Body.Data = postObjectIndex(0)
			i.H["Content-Type"] = []string{plainType}
		}
		return
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
		i.H["Content-Type"] = []string{i.FI[0].ContentType}
		i.H["Content-Disposition"] =
			[]string{attachmentDisposition(i.FI[0].Original)}
		i.L.Body.Data = postObjectIndex(1)
		i.L.Binary = true
		return
	}

	nparts := len(i.FI)
	if hastext {
		nparts++
	}
	xparts := make([]partInfo, nparts)
	x := 0
	if hastext {
		xparts[0].ContentType = plainType
		xparts[0].Body.Data = postObjectIndex(0)
		x++
	}
	for a := range i.FI {
		if len(i.FI[a].ContentType) == 0 {
			panic("Content-Type not set")
		}
		xparts[x].ContentType = i.FI[x].ContentType
		xparts[x].Headers = mail.Headers{
			"Content-Disposition": []string{
				attachmentDisposition(i.FI[x].Original),
			},
		}
		xparts[x].Body.Data = postObjectIndex(1 + a)
		xparts[x].Binary = true
		x++
	}
	i.H["Content-Type"] = []string{"multipart/mixed"}
	i.L.Body.Data = xparts
}
