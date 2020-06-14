package piconfig

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"

	"nksrv/lib/altthumber"
	"nksrv/lib/cacheengine"
	"nksrv/lib/fstore"
	. "nksrv/lib/logx"
	"nksrv/lib/mail/form"
	"nksrv/lib/psql"
	"nksrv/lib/psqlib/internal/pibase"
	"nksrv/lib/psqlib/internal/pibaseweb"
	"nksrv/lib/psqlib/internal/pigpolicy"
	"nksrv/lib/psqlib/internal/pireadnntp"
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
	TCfgPost       *thumbnailer.ThumbConfig
	TCfgOP         *thumbnailer.ThumbConfig
	TCfgSage       *thumbnailer.ThumbConfig
	AltThumber     *altthumber.AltThumber
	WebCaptcha     *webcaptcha.WebCaptcha
	NGPGlobal      string
	NGPAnyPuller   string
	NGPAnyServer   string
	InstanceName   string
}

var stOnce sync.Once

func ConfigPSQLIB(p *pibase.PSQLIB, cfg Config) (err error) {

	stOnce.Do(pibase.LoadStatements)
	if pibase.StLoadErr != nil {
		return pibase.StLoadErr
	}

	p.Log = NewLogToX(*cfg.Logger, fmt.Sprintf("psqlib.%p", p))

	p.DB = *cfg.DB

	err = initDirs(p, cfg)
	if err != nil {
		return
	}

	p.PendingToSrc = fstore.NewMover(&p.Src)
	p.PendingToThm = fstore.NewMover(&p.Thm)

	if cfg.TBuilder != nil {

		p.Thumbnailer, err = cfg.TBuilder.BuildThumbnailer(&p.Thm, *cfg.Logger)
		if err != nil {
			return
		}

		p.ThmPlanForPost = thumbnailer.ThumbPlan{
			Name:        "p",
			ThumbConfig: *cfg.TCfgPost,
		}

		if cfg.TCfgOP != nil {
			p.ThmPlanForOP = thumbnailer.ThumbPlan{
				Name:        "t",
				ThumbConfig: *cfg.TCfgOP,
			}
		} else {
			p.ThmPlanForOP = p.ThmPlanForPost
		}

		if cfg.TCfgSage != nil {
			p.ThmPlanForSage = thumbnailer.ThumbPlan{
				Name:        "s",
				ThumbConfig: *cfg.TCfgSage,
			}
		} else {
			p.ThmPlanForSage = p.ThmPlanForPost
		}

	} else {
		p.Thumbnailer = nilthm.NilThumbnailer{}
	}

	p.NNTPCE = cacheengine.NewCacheEngine(pireadnntp.NNTPCacheMgr{p})

	p.AltThumber = *cfg.AltThumber

	p.FFO = pibase.FormFileOpener{&p.Src}

	p.Instance = nonEmptyStrOrPanic(cfg.NodeName)
	if cfg.WebFrontendKey != "" {
		seed, e := hex.DecodeString(cfg.WebFrontendKey)
		if e != nil {
			panic("bad web frontend key")
		}
		p.WebFrontendKey = ed25519.NewKeyFromSeed(seed)
	}

	p.FPP = form.DefaultParserParams
	// TODO make configurable
	p.FPP.MaxFileCount = 1000
	p.FPP.MaxFileAllSize = 1 << 30

	p.MaxArticleBodySize = (2 << 30) - 1 // TODO config

	p.WebCaptcha = cfg.WebCaptcha
	p.TextPostParamFunc = pibaseweb.MakePostParamFunc(cfg.WebCaptcha)

	p.NGPGlobal, err = pigpolicy.MakeNewGroupPolicy(cfg.NGPGlobal)
	if err != nil {
		return
	}
	p.NGPAnyPuller, err = pigpolicy.MakeNewGroupPolicy(cfg.NGPAnyServer)
	if err != nil {
		return
	}
	p.NGPAnyServer, err = pigpolicy.MakeNewGroupPolicy(cfg.NGPAnyPuller)
	if err != nil {
		return
	}

	p.NTStmts = make(map[int]*sql.Stmt)
	p.NPStmts = make(map[pibase.NPTuple]*sql.Stmt)

	return
}

func nonEmptyStrOrPanic(s string) string {
	if s == "" {
		panic("empty string")
	}
	return s
}
