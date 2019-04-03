package nntpfeedcfg

import (
	"github.com/BurntSushi/toml"

	"centpd/lib/nntp"
)

type parsedBindCfg struct {
	network     string
	addr        string
	listenParam nntp.ListenParam
}

type parsedServer struct {
	BindCfg parsedBindCfg
	RunCfg  nntp.NNTPServerRunCfg
}

type parsedCfg struct {
	servers map[string]parsedServer
	// TODO
}

func ParseCfg(cfg string) (err error) {
	fc := DefaultFeedCfg

	_, err = toml.Decode(cfg, &fc)
	if err != nil {
		return
	}

	// TODO
	return
}
