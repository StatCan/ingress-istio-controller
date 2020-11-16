package controller

func stringInArray(str string, arr []string) bool {
	for _, val := range arr {
		if str == val {
			return true
		}
	}

	return false
}

func stringArrayEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for indx := range a {
		if a[indx] != b[indx] {
			return false
		}
	}

	return true
}
