package webib0

// uses current references in thread and creates backreferences
func ProcessBackReferences(t *IBCommonThread) {
	m := make(map[string]*IBPostInfo)

	m[t.OP.ID] = &t.OP
	for i := range t.Replies {
		m[t.Replies[i].ID] = &t.Replies[i]
	}

	var ii *IBPostInfo
	var rr *IBMessageReference

	ii = &t.OP
	for r := range ii.References {
		rr = &ii.References[r]
		// only process things referencing to current thread
		if rr.Board == "" && rr.Thread == "" && rr.Post != "" {
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
	for i := range t.Replies {
		ii = &t.Replies[i]
		for r := range ii.References {
			rr = &ii.References[r]
			if rr.Board == "" && rr.Thread == "" && rr.Post != "" {
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
}
