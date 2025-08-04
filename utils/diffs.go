package utils

func GetDifferences(current []string, previous []string) ([]string, []string) {
	var added []string
	var removed []string

	for _, c := range current {
		found := false
		for _, p := range previous {
			if c == p {
				found = true
				break
			}
		}
		if !found && c != "" {
			added = append(added, c)
		}
	}

	for _, p := range previous {
		found := false
		for _, c := range current {
			if p == c {
				found = true
				break
			}
		}
		if !found && p != "" {
			removed = append(removed, p)
		}
	}

	return added, removed
}