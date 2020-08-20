package pibase

// psql imageboard module

import (
	"database/sql"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/app/base/altthumber"
	"nksrv/lib/app/base/psql"
	"nksrv/lib/app/base/webcaptcha"
	"nksrv/lib/mail/form"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/utils/fs/cacheengine"
	"nksrv/lib/utils/fs/fstore"
	"nksrv/lib/utils/logx"

	"nksrv/lib/app/psqlib/internal/pigpolicy"
)

type PSQLIB struct {
	// database handle
	DB psql.PSQL
	// this psqlib instance logger
	Log logx.LogToX
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
