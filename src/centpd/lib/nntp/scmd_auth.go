package nntp

import . "centpd/lib/logx"

func cmdAuthInfo(c *ConnState, args [][]byte, rest []byte) bool {
	args = args[:0] // reuse

	if len(rest) == 0 {
		AbortOnErr(c.w.PrintfLine("501 AUTHINFO keyword expected"))
		return true
	}

	x := parseKeyword(rest)

	cmd, ok := authCommandMap[string(rest[:x])]
	if !ok {
		AbortOnErr(c.w.PrintfLine("501 unrecognised AUTHINFO keyword"))
		return true
	}

	if x >= len(rest) {
		goto argsparsed
	}

	for {
		// skip spaces
		for {
			x++
			if x >= len(rest) {
				goto argsparsed
			}
			if rest[x] != ' ' && rest[x] != '\t' {
				break
			}
		}
		if len(args) >= cmd.maxargs {
			if !cmd.allowextra {
				AbortOnErr(c.w.PrintfLine("501 too much parameters"))
			} else {
				cmd.cmdfunc(c, args, rest[x:])
			}
			return true
		}
		sx := x
		// skip non-spaces
		for {
			x++
			if x >= len(rest) {
				args = append(args, rest[sx:x])
				goto argsparsed
			}
			if rest[x] == ' ' || rest[x] == '\t' {
				args = append(args, rest[sx:x])
				break
			}
		}
	}
argsparsed:
	if len(args) < cmd.minargs {
		AbortOnErr(c.w.PrintfLine("501 not enough parameters"))
		return true
	}
	cmd.cmdfunc(c, args, nil)
	return true
}

func authCmdUser(c *ConnState, args [][]byte, rest []byte) bool {
	ol := c.pullActiveLogin()
	if ol != nil {
		AbortOnErr(c.w.PrintfLine(
			"482 authentication commands issued out of sequence"))
		return true
	}
	if c.authenticated {
		// do not allow multiple authentications
		AbortOnErr(c.w.PrintfLine("502 command unavailable"))
		return true
	}
	rCfg := c.srv.GetRunCfg()
	if rCfg.UserPassProvider == nil {
		AbortOnErr(c.w.PrintfLine("503 AUTHINFO USER unimplemented"))
		return true
	}
	if !rCfg.UnsafePass && !c.tlsStarted() {
		AbortOnErr(c.w.PrintfLine("483 TLS required"))
		return true
	}
	ui, ch := rCfg.UserPassProvider.NNTPUserPassByName(unsafeBytesToStr(args[0]))
	// prevent user enumeration by default
	// XXX this isn't constant-time but should be fine still
	if rCfg.UnsafeEarlyUserReject && ui == nil {
		AbortOnErr(c.w.PrintfLine("481 authentication failed"))
		return true
	}
	// I don't see issue accepting passwordless users early though
	if ch == "" && ui != nil {
		AbortOnErr(c.w.PrintfLine("281 authentication accepted"))
		c.loginSuccess(ui)
		return true
	}
	// otherwise require pass
	AbortOnErr(c.w.PrintfLine("381 password required"))
	c.activeLogin = &ActiveLogin{ui: ui, ch: ch}
	return true
}

func authCmdPass(c *ConnState, args [][]byte, rest []byte) bool {
	ol := c.pullActiveLogin()
	if ol == nil {
		// send some kind of rejection. but WHAT kind?
		if c.authenticated {
			// do not allow multiple authentications
			AbortOnErr(c.w.PrintfLine("502 command unavailable"))
			return true
		}
		rCfg := c.srv.GetRunCfg()
		if rCfg.UserPassProvider == nil {
			AbortOnErr(c.w.PrintfLine("503 AUTHINFO PASS unimplemented"))
			return true
		}
		if !rCfg.UnsafePass && !c.tlsStarted() {
			AbortOnErr(c.w.PrintfLine("483 TLS required"))
			return true
		}
		AbortOnErr(c.w.PrintfLine(
			"482 authentication commands issued out of sequence"))
		return true
	}

	upp := c.srv.GetRunCfg().UserPassProvider
	rpass := ClientPassword(unsafeBytesToStr(args[0]))
	if ol.ui == nil ||
		(ol.ch != "" && (upp == nil || !upp.NNTPCheckPass(ol.ch, rpass))) {

		AbortOnErr(c.w.PrintfLine("481 authentication failed"))
		return true
	}

	AbortOnErr(c.w.PrintfLine("281 authentication accepted"))
	c.loginSuccess(ol.ui)
	return true
}

func (c *ConnState) loginSuccess(ui *UserInfo) {
	c.authenticated = true
	c.UserPriv = MergeUserPriv(c.UserPriv, ui.UserPriv)
	c.log.LogPrintf(NOTICE, "logged in as name=%q serv=%q", ui.Name, ui.Serv)
}

func authCmdSASL(c *ConnState, args [][]byte, rest []byte) bool {
	ToUpperASCII(args[0])
	// TODO
	AbortOnErr(c.w.PrintfLine("503 AUTHINFO SASL %s unimplemented", args[0]))
	return true
}
