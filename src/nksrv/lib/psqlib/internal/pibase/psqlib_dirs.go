package pibase

// psqlib directory initializations

import (
	"nksrv/lib/fstore"
)

const (
	PendingDir          = "pending" // for src & thm
	NNTPIncomingTempDir = "in_tmp"
	NNTPIncomingDir     = "in_got"
	NNTPPullerDir       = "in_pulled"
)

func (p *PSQLIB) initDirs(cfg Config) (err error) {
	p.Src, err = fstore.OpenFStore(*cfg.SrcCfg)
	if err != nil {
		return
	}
	err = p.Src.DeclareDir("tmp", false)
	if err != nil {
		return
	}

	p.Thm, err = fstore.OpenFStore(*cfg.ThmCfg)
	if err != nil {
		return
	}
	err = p.Thm.DeclareDir("tmp", false)
	if err != nil {
		return
	}

	p.NNTPFS, err = fstore.OpenFStore(*cfg.NNTPFSCfg)
	if err != nil {
		return
	}
	err = p.NNTPFS.DeclareDir(NNTPIncomingTempDir, false)
	if err != nil {
		return
	}
	err = p.NNTPFS.DeclareDir(NNTPIncomingDir, true)
	if err != nil {
		return
	}
	err = p.NNTPFS.DeclareDir(NNTPPullerDir, true)
	if err != nil {
		return
	}

	return
}
