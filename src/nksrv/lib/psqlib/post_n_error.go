package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	au "nksrv/lib/asciiutils"
	"nksrv/lib/date"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"
	"nksrv/lib/mailibsign"
	"nksrv/lib/nntp"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
)

// system error, not message being faulty
type NNTPMessageError struct {
	x string
}
