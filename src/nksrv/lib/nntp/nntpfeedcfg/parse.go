package nntpfeedcfg

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"

	"nksrv/lib/nntp"
	"nksrv/lib/utils/certfp"
)

type parsedUserCfg struct {
	ui nntp.UserInfo
	ch string
}

type parsedCertFPCfg struct {
	ui nntp.UserInfo
	sl certfp.Selector
	fp string
}

type parsedBindCfg struct {
	network     string
	addr        string
	listenParam nntp.ListenParam
}

type parsedServerParams struct {
	nntps                 bool
	cert                  tls.Certificate
	priv                  nntp.UserPriv
	tlspriv               nntp.UserPriv
	unsafePass            bool
	unsafeEarlyUserReject bool
	certfpAutoAuth        bool
}

type parsedServer struct {
	BindCfg []parsedBindCfg

	parsedServerParams
}

type parsedCfg struct {
	users   []parsedUserCfg
	certfp  []parsedCertFPCfg
	servers map[string]parsedServer
	peers   map[string]parsedPeer
	// TODO
}

func ParseCfg(cfg string) (err error) {
	var pcfg parsedCfg
	fc := DefaultFeedCfg

	md, err = toml.Decode(cfg, &fc)
	if err != nil {
		return
	}

	for i := range fc.Users {
		// default priv
		fc_user := UserCfg{Priv: fc.UsersPriv}

		err = md.PrimitiveDecode(fc.Users[i], &fc_user)
		if err != nil {
			return
		}
		priv, ok := nntp.ParseUserPriv(fc_user.Priv, nntp.UserPriv{})
		if !ok {
			return fmt.Errorf("unrecognised priv %q", fc_user.Priv)
		}
		pcfg.users = append(pcfg.users, parsedUserCfg{
			ui: nntp.UserInfo{Name: fc_user.Name, UserPriv: priv},
			ch: fc_user.Pass,
		})
	}
	for i := range fc.CertFP {
		// default priv
		fc_certfp := CertFPCfg{Priv: fc.CertFPPriv}

		err = md.PrimitiveDecode(fc.CertFP[i], &fc_certfp)
		if err != nil {
			return
		}
		priv, ok := nntp.ParseUserPriv(fc_certfp.Priv, nntp.UserPriv{})
		if !ok {
			return fmt.Errorf("unrecognised priv %q", fc_certfp.Priv)
		}
		pcfp := parsedCertFPCfg{
			ui: nntp.UserInfo{Name: fc_user.Name, UserPriv: priv},
		}
		if fc_certfp.Cert != "" {
			pcfp.sl = certfp.SelectorFull
			pcfp.fp = fc_certfp.Cert
		} else if fc_certfp.PubKey != "" {
			pcfp.sl = certfp.SelectorPubKey
			pcfp.fp = fc_certfp.PubKey
		} else {
			return errors.New("either cert or pubkey must be set")
		}
		pcfg.certfp = append(pcfg.certfp, pcfp)
	}

	//parseServAddr := func(listen toml.Primitive) []
	for i := range fc.Servers {
		fc_server := ServerCfg{ServerInnerCfg: fc.ServersDefault}

		err = md.PrimitiveDecode(fc.Servers[i], &fc_server)
		if err != nil {
			return
		}

		listenstrs := make([]string, 1)
		err = md.PrimitiveDecode(fc_server.Listen, &listenstrs[0])
		if err != nil {
			err = md.PrimitiveDecode(fc_server.Listen, &listenstrs)
			if err != nil {
				return
			}
		}
		//if len(listenstrs) == 0 {

	}

	// TODO
	return
}
