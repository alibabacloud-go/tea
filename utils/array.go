package utils

func Contains(s []*string, str *string) bool {
	if s == nil {
		return false
	}
	for _, v := range s {
		if str != nil && v != nil && *v == *str {
			return true
		}
	}
	return false
}
