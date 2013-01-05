package main

// isDigit returns true if the rune d represents an ascii digit
// between 0 and 9, inclusive.
func isDigit(d uint8) bool {
	return d >= '0' && d <= '9'
}

// splitSpaces returns a slice of byte slices which are the space
// delimited words from the original byte slice.  Unlike
// strings.Split($X, " "), runs of multiple spaces in a row are
// discarded.  NOTE WELL: this only checks for spaces (' '), other
// unicode whitespace isn't supported.
func splitSpaces(b []byte) [][]byte {
	// most lines in smaps have the form "Swap: 4 kB", so
	// preallocate the slice's array appropriately.
	res := make([][]byte, 0, 3)
	start := 0
	for i := 0; i < len(b)-1; i++ {
		if b[i] == ' ' {
			start = i + 1
		} else if b[i+1] == ' ' {
			res = append(res, b[start:i+1])
			start = i + 1
		}
	}
	if start != len(b) && b[start] != ' ' {
		res = append(res, b[start:])
	}
	return res
}
