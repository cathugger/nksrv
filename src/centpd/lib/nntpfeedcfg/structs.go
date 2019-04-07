package nntpfeedcfg

import "github.com/BurntSushi/toml"

type UserInnerCfg struct {
	Name string `toml:"name"`
	Pass string `toml:"pass"`
}

type UserCfg struct {
	UserInnerCfg

	Priv string `toml:"priv"`
}

type CertFPInnerCfg struct {
	Cert   string `toml:"cert"`
	PubKey string `toml:"pubkey"`
}

type CertFPCfg struct {
	CertFPInnerCfg

	Priv string `toml:"priv"`
	Name string `toml:"name"`
}

type PrivCertCfg struct {
	Certificate string `toml:"cert"`
	PrivateKey  string `toml:"priv"`
}

type ServerInnerCfg struct {
	Enabled    bool        `toml:"enabled"`
	Priv       string      `toml:"priv"`
	TLSCert    PrivCertCfg `toml:"tls_cert"`
	UnsafePass bool        `toml:"unsafepass"`
	// TODO
}

var DefaultServerInnerCfg = ServerInnerCfg{
	Enabled: true,
	Priv:    "rw",
}

type ServerCfg struct {
	ServerInnerCfg

	Listen toml.Primitive `toml:"listen"`
	NNTPS  bool           `toml:"tls_nntps"`
	// TODO
}

type PeerInnerCfg struct {
	Enabled     bool        `toml:"enabled"`
	DialCert    PrivCertCfg `toml:"dial_cert"`
	Pull        bool        `toml:"pull"`
	PullWorkers int         `toml:"pull_workers"`
	Push        bool        `toml:"push"`
	PushWorkers int         `toml:"push_workers"`
	ServPriv    string      `toml:"serv_priv"`
}

var DefaultPeerInnerCfg = PeerInnerCfg{
	Enabled:  true,
	Pull:     true,
	ServPriv: "rw",
}

type PeerCfg struct {
	PeerInnerCfg

	DialAddr   string       `toml:"dial"`
	DialUser   UserInnerCfg `toml:"dial_user"` // sharing this would be bad idea
	ServUser   UserCfg      `toml:"serv_user"`
	ServCertFP CertFPCfg    `toml:"serv_certfp"`
}

type FeedCfg struct {
	Users     []toml.Primitive `toml:"users"`
	UsersPriv string           `toml:"users_priv"` // default priv if not specified

	CertFP     []toml.Primitive `toml:"certfp"`
	CertFPPriv string           `toml:"certfp_priv"`

	ServersDefault ServerInnerCfg            `toml:"servers_all"`
	Servers        map[string]toml.Primitive `toml:"servers"`

	PeersDefault PeerInnerCfg              `toml:"peers_all"`
	Peers        map[string]toml.Primitive `toml:"peers"`
}

var DefaultFeedCfg = FeedCfg{
	UsersPriv:      "rw",
	CertFPPriv:     "rw",
	ServersDefault: DefaultServerInnerCfg,
	PeersDefault:   DefaultPeerInnerCfg,
}
