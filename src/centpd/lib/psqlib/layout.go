package psqlib

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"strings"

	"golang.org/x/crypto/ed25519"

	au "centpd/lib/asciiutils"
	"centpd/lib/ftypes"
	ht "centpd/lib/hashtools"
	. "centpd/lib/logx"
	"centpd/lib/mail"
	"centpd/lib/mail/form"
	"centpd/lib/mailib"
)

func attachmentDisposition(oname string) (cdis string) {
	cdis = mail.FormatMediaTypeX(
		"attachment", map[string]string{"filename": oname})
	return
}

func attachmentConentType(ctype string, oname string) string {
	ct, cpar, err := mime.ParseMediaType(ctype)
	if err != nil {
		// cannot parse media type -- cannot add "name" parameter
		return ctype
	}

	// always escape using bencoding if needed, for compat,
	// proper readers will use disposition anyway
	cpar["name"] = mime.BEncoding.Encode("UTF-8", oname)
	return mail.FormatMediaTypeX(ct, cpar)
}

const (
	plainTextType = "text/plain; charset=US-ASCII"
	plainUTF8Type = "text/plain; charset=UTF-8"
	messageType   = "message/rfc822"
)

type FormFileList struct {
	files map[string][]form.File
}

func (ffl FormFileList) OpenFileAt(i int) (io.ReadCloser, error) {
	n := 0
	// yeah this is stupid
	for _, fieldname := range FileFields {
		files := ffl.files[fieldname]
		for i := range files {
			fn := files[i].F.Name()
			if i == n {
				return os.Open(fn)
			}

			n++
		}
	}
	panic("out of bounds")
}

func tohex(b []byte) string {
	return hex.EncodeToString(b)
}

func (sp *PSQLIB) fillWebPostDetails(
	i mailib.PostInfo, frm form.Form, board string,
	ref CoreMsgIDStr, inreplyto []string, signkeyseed []byte) (_ mailib.PostInfo, mfn string, err error) {

	i = sp.fillWebPostInner(i, board, ref, inreplyto)

	if len(signkeyseed) != 0 {

		// signing (this is painful to do :<)

		seckey := ed25519.NewKeyFromSeed(signkeyseed)
		pubkey := seckey[32:64]

		// we need to first generate inner message then sign it

		// need to include subject now
		if i.MI.Title != "" {
			i.H["Subject"] = mail.OneHeaderVal(i.MI.Title)
		}

		// new file for message
		f, e := sp.ffo.OpenFile()
		if e != nil {
			err = fmt.Errorf("err opening message file: %v", e)
			return
		}
		mfn = f.Name()
		defer func() {
			// cleanup if err
			if err != nil {
				f.Close()
				os.Remove(mfn)
				mfn = ""
			}
		}()

		// do generation itself
		e = mailib.GenerateMessage(f, i, FormFileList{files: frm.Files})
		if e != nil {
			err = fmt.Errorf("err generating inner message: %v", e)
			return
		}

		// seek to start
		_, e = f.Seek(0, 0)
		if e != nil {
			err = fmt.Errorf("err seeking message file: %v", e)
			return
		}

		// perform sign hashing and examination of generated message
		signhasher := sha512.New()
		trdr := &mailib.ReadTracker{R: au.NewUnixTextReader(f)}
		_, e = io.Copy(signhasher, trdr)
		if e != nil {
			err = fmt.Errorf("err hashing message: %v", e)
			return
		}
		var signhashbuf [64]byte
		signhash := signhasher.Sum(signhashbuf[:0])

		// we need to know precise file size (unixtextreader may obscure it)
		n, e := f.Seek(0, 2)
		if e != nil {
			err = fmt.Errorf("err seeking2 message file: %v", e)
			return
		}

		// seek to begin again for filename hashing
		_, e = f.Seek(0, 0)
		if e != nil {
			err = fmt.Errorf("err seeking3 message file: %v", e)
			return
		}

		// hash content, produce filename
		hash, hashtype, e := ht.MakeFileHash(f)
		if e != nil {
			err = fmt.Errorf("err making message filehash: %v", e)
			return
		}

		// we're done with this file so close
		e = f.Close()
		if e != nil {
			err = fmt.Errorf("err closing message file: %v", e)
			return
		}

		// specify minimal file info
		mfi := mailib.FileInfo{
			Type:        ftypes.FTypeMsg,
			ContentType: messageType,
			Size:        n,
			ID:          hash + "-" + hashtype + ".eml",
		}
		// add it
		i.FI = append(i.FI, mfi)

		// generate new layout
		i.L = mailib.PartInfo{}
		i.L.Body.Data = mailib.PostObjectIndex(len(i.FI))
		i.L.HasNull = trdr.HasNull
		i.L.Has8Bit = trdr.Has8Bit && !trdr.HasNull

		// cleanup headers
		delete(i.H, "Subject")
		for k := range i.H {
			if strings.HasPrefix(k, "Content-") {
				delete(i.H, k)
			}
		}

		// add new headers we need
		i.H["MIME-Version"] = mail.OneHeaderVal("1.0")
		i.H["Content-Type"] = mail.OneHeaderVal(messageType)

		// sign
		sig := ed25519.Sign(seckey, signhash)
		pubkeystr := tohex(pubkey)
		sigstr := tohex(sig)
		sp.log.LogPrintf(DEBUG, "msgsig: hash %X pubkey %s signature %s", signhash, pubkeystr, sigstr)

		i.MI.Trip = pubkeystr // write tripcode
		i.H["X-PubKey-Ed25519"] = mail.OneHeaderVal(pubkeystr)
		i.H["X-Signature-Ed25519-SHA512"] = mail.OneHeaderVal(sigstr)
	}

	// Path
	i.H["Path"] = mail.OneHeaderVal(sp.instance + "!.POSTED!not-for-mail")

	return i, mfn, nil
}

func (sp *PSQLIB) fillWebPostInner(
	i mailib.PostInfo, board string,
	ref CoreMsgIDStr, inreplyto []string) mailib.PostInfo {

	hastext := len(i.MI.Message) != 0
	text8bit := !au.Is7BitString(i.MI.Message)

	if i.H != nil {
		panic("header should be nil at this point")
	}

	i.H = make(mail.Headers)

	// we don't really need to store Message-ID there

	// we usually don't need to store Subject there if MI.Title is valid
	// Pan (that trash gnome newsreader) don't list articles with no Subject. this behavior is outright batshit insane if you ask me.
	// Thunderbird lists such articles but with no replacement text and it looks kinda bad but pretty usable still.
	// Sylpheed and Claws get it right, adding replacement text.
	// standards for Usenet seems to require non-empty Subject value.
	// seems it would be ok-ish idea to include None here for compatibility but I fucking hate pointless "None" in articles just for this
	// I'll rather add such hack in OVER/XOVER (as that'll make existing articles work)
	/*
		if i.MI.Title == "" {
			i.H["Subject"] = mail.OneHeaderVal("None")
		}
	*/

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

	// In-Reply-To
	if len(inreplyto) != 0 {
		if ref == "" && len(inreplyto) == 1 {
			// add dummy to prevent misinterpretation by
			// standards compliant clients
			inreplyto = append(inreplyto, "<0>")
		}
		i.H["In-Reply-To"] = mail.OneHeaderVal(strings.Join(inreplyto, " "))
	}

	// X-Sage
	if i.MI.Sage && ref != "" {
		// NOTE: some impls specifically check for "1"
		i.H["X-Sage"] = mail.OneHeaderVal("1")
	}

	// now deal with layout

	if len(i.FI) == 0 {
		if !hastext {
			// empty. don't include Content-Type header either
			i.L.Body.Data = nil
		} else {
			i.L.Body.Data = mailib.PostObjectIndex(0)
			if !text8bit {
				i.H["Content-Type"] = mail.OneHeaderVal(plainTextType)
			} else {
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
		if !text8bit {
			// workaround for nntpchan before 2018-12-23
			xparts[0].ContentType = plainTextType
		} else {
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
