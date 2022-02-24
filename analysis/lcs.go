package analysis

// LCS uses largest common subsequence and returns that sequence.
func LCS(X []string, Y []string, l, k int) ([]string, int) {
	if l == 0 || k == 0 {
		return []string{}, 0
	}
	if X[l-1] == Y[k-1] {
		a, b := LCS(X, Y, l-1, k-1)
		return append(a, X[l-1]), b + 1
	}
	ax, bx := LCS(X, Y, l-1, k)
	ay, by := LCS(X, Y, l, k-1)
	if bx >= by {
		return ax, bx
	}
	return ay, by
}

// LCSI uses largest common subsequence algorithm and returns
// two slices of indices of two inputs that create the largest
// common subsequence between the two.
func LCSI(X []string, Y []string, l, k int) ([]int, []int, int) {
	if l == 0 || k == 0 {
		return []int{}, []int{}, 0
	}
	if X[l-1] == Y[k-1] {
		a1, a2, b := LCSI(X, Y, l-1, k-1)
		return append(a1, l-1), append(a2, k-1), b + 1
	}
	ax1, ax2, bx := LCSI(X, Y, l-1, k)
	ay1, ay2, by := LCSI(X, Y, l, k-1)
	if bx >= by {
		return ax1, ax2, bx
	}
	return ay1, ay2, by
}

type LCSIM_struct struct {
	Left  []int
	Right []int
	Len   int
}

var (
	LCSIM_lru map[int]map[int]LCSIM_struct
)

func LCSIM(X []string, Y []string) LCSIM_struct {
	LCSIM_lru = make(map[int]map[int]LCSIM_struct)
	for i := 0; i <= len(X); i++ {
		LCSIM_lru[i] = make(map[int]LCSIM_struct)
	}
	ret := LCSIM_r(X, Y, len(X), len(Y))
	// Clear the LRU cache
	LCSIM_lru = nil
	return ret
}

func LCSIM_r(X []string, Y []string, l, k int) LCSIM_struct {
	if l == 0 || k == 0 {
		return LCSIM_struct{
			Left:  []int{},
			Right: []int{},
			Len:   0,
		}
	}
	if v, ok := LCSIM_lru[l][k]; ok {
		return v
	}
	if X[l-1] == Y[k-1] {
		st := LCSIM_r(X, Y, l-1, k-1)
		toSave := LCSIM_struct{
			Left:  append(st.Left, l-1),
			Right: append(st.Right, k-1),
			Len:   st.Len + 1,
		}
		LCSIM_lru[l][k] = toSave
		return toSave
	}
	st1 := LCSIM_r(X, Y, l-1, k)
	st2 := LCSIM_r(X, Y, l, k-1)
	if st1.Len >= st2.Len {
		LCSIM_lru[l][k] = st1
		return st1
	}
	LCSIM_lru[l][k] = st2
	return st2
}
