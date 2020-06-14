package pibase

// psql imageboard module

import (
	"database/sql"
	"sync"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/altthumber"
	"nksrv/lib/cacheengine"
	"nksrv/lib/fstore"
	. "nksrv/lib/logx"
	"nksrv/lib/mail/form"
	"nksrv/lib/psql"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/webcaptcha"

	"nksrv/lib/psqlib/internal/pigpolicy"
)

type TBoardID = uint32
type TPostID = uint64

type NPTuple struct {
	N    int
	Sage bool
}

const (
	PendingDir          = "pending" // for src & thm
	NNTPIncomingTempDir = "in_tmp"
	NNTPIncomingDir     = "in_got"
	NNTPPullerDir       = "in_pulled"
)

type PSQLIB struct {
	// database handle
	DB psql.PSQL
	// this psqlib instance logger
	Log LogToX
	// file storage stuff
	Src    fstore.FStore
	Thm    fstore.FStore
	NNTPFS fstore.FStore
	// movers for posting
	PendingToSrc fstore.Mover
	PendingToThm fstore.Mover
	// for caching of generated NetNews articles
	NNTPCE cacheengine.CacheEngine
	// thumbnailing things
	Thumbnailer    thumbnailer.Thumbnailer
	ThmPlanForPost thumbnailer.ThumbPlan
	ThmPlanForOP   thumbnailer.ThumbPlan
	ThmPlanForSage thumbnailer.ThumbPlan
	AltThumber     altthumber.AltThumber
	// HTTP POST form handling
	FFO               FormFileOpener
	FPP               form.ParserParams
	TextPostParamFunc func(string) bool

	Instance           string
	MaxArticleBodySize int64
	WebCaptcha         *webcaptcha.WebCaptcha
	WebFrontendKey     ed25519.PrivateKey

	NGPGlobal    pigpolicy.NewGroupPolicy
	NGPAnyPuller pigpolicy.NewGroupPolicy
	NGPAnyServer pigpolicy.NewGroupPolicy

	StPrep [stMax]*sql.Stmt

	// newthread prepared statements and locking
	NTStmts map[int]*sql.Stmt
	NTMutex sync.RWMutex

	// newpost prepared statements and locking
	NPStmts map[NPTuple]*sql.Stmt
	NPMutex sync.RWMutex

	PullerNonce int64

	NoFileSync bool
}

// readonly for now

func (sp *PSQLIB) Prepare() (err error) {
	err = sp.prepareStatements()
	if err != nil {
		return
	}

	return
}

func (sp *PSQLIB) Close() error {
	return sp.closeStatements()
}
