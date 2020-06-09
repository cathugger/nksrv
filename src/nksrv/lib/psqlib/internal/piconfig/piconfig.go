package piconfig

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"fmt"

	"nksrv/lib/altthumber"
	"nksrv/lib/cacheengine"
	"nksrv/lib/fstore"
	"nksrv/lib/mail/form"
	"nksrv/lib/psql"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/nilthm"
	"nksrv/lib/webcaptcha"
)

type Config struct {
	DB             *psql.PSQL
	Logger         *LoggerX
	NodeName       string
	WebFrontendKey string
	SrcCfg         *fstore.Config
	ThmCfg         *fstore.Config
	NNTPFSCfg      *fstore.Config
	TBuilder       thumbnailer.ThumbnailerBuilder
	TCfgThread     *thumbnailer.ThumbConfig
	TCfgReply      *thumbnailer.ThumbConfig
	TCfgSage       *thumbnailer.ThumbConfig
	AltThumber     *altthumber.AltThumber
	WebCaptcha     *webcaptcha.WebCaptcha
	NGPGlobal      string
	NGPAnyPuller   string
	NGPAnyServer   string
	InstanceName   string
}

func NewPSQLIB(cfg Config) (p *PSQLIB, err error) {
	p = new(PSQLIB)

	st_once.Do(loadStatements)
	if st_loaderr != nil {
		return nil, st_loaderr
	}

	p.log = NewLogToX(*cfg.Logger, fmt.Sprintf("psqlib.%p", p))

	p.db = *cfg.DB

	err = p.initDirs(cfg)
	if err != nil {
		return nil, err
	}

	p.pending2src = fstore.NewMover(&p.src)
	p.pending2thm = fstore.NewMover(&p.thm)

	if cfg.TBuilder != nil {

		p.thumbnailer, err = cfg.TBuilder.BuildThumbnailer(&p.thm, *cfg.Logger)
		if err != nil {
			return nil, err
		}

		p.tplan_thread = thumbnailer.ThumbPlan{
			Name:        "t",
			ThumbConfig: *cfg.TCfgThread,
		}
		p.tplan_reply = thumbnailer.ThumbPlan{
			Name:        "r",
			ThumbConfig: *cfg.TCfgReply,
		}
		if cfg.TCfgSage != nil {
			p.tplan_sage = thumbnailer.ThumbPlan{
				Name:        "s",
				ThumbConfig: *cfg.TCfgSage,
			}
		} else {
			p.tplan_sage = p.tplan_reply
		}

	} else {
		p.thumbnailer = nilthm.NilThumbnailer{}
	}

	p.nntpce = cacheengine.NewCacheEngine(nntpcachemgr{p})

	p.altthumb = *cfg.AltThumber

	p.ffo = formFileOpener{&p.src}

	p.instance = nonEmptyStrOrPanic(cfg.NodeName)
	if cfg.WebFrontendKey != "" {
		seed, e := hex.DecodeString(cfg.WebFrontendKey)
		if e != nil {
			panic("bad web frontend key")
		}
		p.webFrontendKey = ed25519.NewKeyFromSeed(seed)
	}

	p.fpp = form.DefaultParserParams
	// TODO make configurable
	p.fpp.MaxFileCount = 1000
	p.fpp.MaxFileAllSize = 1 << 30

	p.maxArticleBodySize = (2 << 30) - 1 // TODO config

	p.webcaptcha = cfg.WebCaptcha
	p.textPostParamFunc = makePostParamFunc(cfg.WebCaptcha)

	p.ngp_global, err = makeNewGroupPolicy(cfg.NGPGlobal)
	if err != nil {
		return
	}
	p.ngp_anyserver, err = makeNewGroupPolicy(cfg.NGPAnyServer)
	if err != nil {
		return
	}
	p.ngp_anyserver, err = makeNewGroupPolicy(cfg.NGPAnyPuller)
	if err != nil {
		return
	}

	p.ntStmts = make(map[int]*sql.Stmt)
	p.npStmts = make(map[npTuple]*sql.Stmt)

	return
}

func nonEmptyStrOrPanic(s string) string {
	if s == "" {
		panic("empty string")
	}
	return s
}
