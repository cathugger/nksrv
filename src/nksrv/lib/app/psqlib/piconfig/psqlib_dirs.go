package piconfig

// psqlib directory initializations

import (
	"nksrv/lib/app/psqlib/internal/pibase"
	"nksrv/lib/utils/fs/fstore"
)

func initDirs(p *pibase.PSQLIB, cfg Config) (err error) {
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
	err = p.NNTPFS.DeclareDir(pibase.NNTPIncomingTempDir, false)
	if err != nil {
		return
	}
	err = p.NNTPFS.DeclareDir(pibase.NNTPIncomingDir, true)
	if err != nil {
		return
	}
	err = p.NNTPFS.DeclareDir(pibase.NNTPPullerDir, true)
	if err != nil {
		return
	}

	return
}
