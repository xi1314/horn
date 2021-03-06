package utils

func isSid(uid string) bool {
	return len(uid) == 19
}

func isVid(uid string) bool {
	return len(uid) == 23
}

func GetRole(uid string) string {
	if isSid(uid) {
		return "staff"
	} else {
		return "visitor"
	}
}
