package psqlib

// psqlib directory initializations

import (
	"nksrv/lib/fstore"
)

const (
	pendingDir          = "pending" // for src & thm
	nntpIncomingTempDir = "in_tmp"
	nntpIncomingDir     = "in_got"
	nntpPullerDir       = "in_pulled"
)

func (p *PSQLIB) initDirs(cfg Config) (err error) {
	p.src, err = fstore.OpenFStore(*cfg.SrcCfg)
	if err != nil {
		return
	}
	err = p.src.DeclareDir("tmp", false)
	if err != nil {
		return
	}

	p.thm, err = fstore.OpenFStore(*cfg.ThmCfg)
	if err != nil {
		return
	}
	err = p.thm.DeclareDir("tmp", false)
	if err != nil {
		return
	}

	p.nntpfs, err = fstore.OpenFStore(*cfg.NNTPFSCfg)
	if err != nil {
		return
	}
	err = p.nntpfs.DeclareDir(nntpIncomingTempDir, false)
	if err != nil {
		return
	}
	err = p.nntpfs.DeclareDir(nntpIncomingDir, true)
	if err != nil {
		return
	}
	err = p.nntpfs.DeclareDir(nntpPullerDir, true)
	if err != nil {
		return
	}

	return
}
