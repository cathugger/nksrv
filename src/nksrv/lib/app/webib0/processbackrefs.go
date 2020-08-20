package webib0

// uses current references in thread and creates backreferences
func ProcessBackReferences(bn string, t *IBCommonThread) {
	m := make(map[string]*IBPostInfo)

	m[t.OP.ID] = &t.OP
	for i := range t.Replies {
		m[t.Replies[i].ID] = &t.Replies[i]
	}

	procrefs := func(ii *IBPostInfo) {
		for r := range ii.References {
			rr := &ii.References[r]
			// only process things referencing to current thread
			if (rr.Board == "" || rr.Board == bn) &&
				(rr.Thread == "" || rr.Thread == t.ID) && rr.Post != "" {

				// self-references are skipped
				if p := m[rr.Post]; p != nil && p != ii {

					br := IBBackReference{IBReference{Post: ii.ID}}
					// doubles are skipped too
					if len(p.BackReferences) != 0 &&
						p.BackReferences[len(p.BackReferences)-1] == br {

						continue
					}

					p.BackReferences = append(p.BackReferences, br)
				}
			}
		}
	}

	procrefs(&t.OP)
	for i := range t.Replies {
		procrefs(&t.Replies[i])
	}
}
