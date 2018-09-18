package psqlib

import (
	"fmt"
	"io"
	"time"
)

func (sp *PSQLIB) nntpGenerate(w io.Writer, num uint64, msgid CoreMsgIDStr) error {
	// TODO
	// need to generate article off database shit
	// im too fucking lazy to actually do it atm
	// so placeholder to test shit will work for now
	fmt.Fprintf(w, "Message-ID: <%s>\n\n", string(msgid))
	for i := 0; i < 20; i++ {
		time.Sleep(250 * time.Millisecond)
		fmt.Fprintf(w, "faggot\n")
	}
	return nil
}
