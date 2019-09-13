package history

type TektonPull struct {
	Number int
	Title  string
	SHA    string
	Author string
}

// ByNum implements sort.Interface for []PRMeta to sort by ascending PR number.
type ByNum []TektonPull

func (prs ByNum) Len() int           { return len(prs) }
func (prs ByNum) Swap(i, j int)      { prs[i], prs[j] = prs[j], prs[i] }
func (prs ByNum) Less(i, j int) bool { return prs[i].Number < prs[j].Number }
