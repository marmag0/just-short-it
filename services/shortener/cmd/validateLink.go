package main

func initialLinkValid(link string) (bool, string) {
	if len(link) < 3 {
		return false, "short link's length is too short, min length is 3"
	} else if len(link) > 32 {
		return false, "short link's length is too long, max lenght is 32"
	}

	for i := 0; i < len(link); i++ {
		c := link[i]
		isAlphaNum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlphaNum {
			return false, "short link contain disallowed characters"
		}
	}

	return true, ""
}
