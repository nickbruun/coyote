package errorhandlers

func isAscii(str string) bool {
	for _, c := range str {
		if c <= 31 || c >= 127 {
			return false
		}
	}

	return true
}
