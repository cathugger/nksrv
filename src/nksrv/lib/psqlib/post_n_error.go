package psqlib

import (
	. "nksrv/lib/logx"
)

// system error, not message being faulty
type NNTPMessageError struct {
	x string
}
