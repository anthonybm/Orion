package util

// AppendToDoubleSlice appends all string slices from entries to dest and returns a new object holding the slice of string slices
func AppendToDoubleSlice(dest [][]string, entries [][]string) (newdest [][]string) {
	for _, v := range dest {
		newdest = append(newdest, v)
	}
	for _, v := range entries {
		newdest = append(newdest, v)
	}
	return
}
