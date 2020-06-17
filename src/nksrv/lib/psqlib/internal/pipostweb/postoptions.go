package pipostweb

import "strings"

type PostOptions struct {
	sage    bool
	nolimit bool
}

func parsePostOptions(opts string) (ok bool, popts PostOptions) {
	sopts := strings.Split(opts, ",")
	for i, sopt := range sopts {
		topt := strings.ToLower(strings.TrimSpace(sopt))
		switch topt {
		case "sage":
			popts.sage = true
		case "nolimit":
			popts.nolimit = true
		case "":
			if i != len(sopts)-1 {
				return
			}
		default:
			// idk this opt, not ok
			return
		}
	}
	// all gucci
	ok = true
	return
}
