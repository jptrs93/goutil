package envu

// IsValidPOSIXEnvKey reports whether key is a valid POSIX shell environment variable name.
func IsValidPOSIXEnvKey(key string) bool {
	if key == "" {
		return false
	}

	for i := 0; i < len(key); i++ {
		c := key[i]
		if c == '_' || ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') {
			continue
		}
		if i > 0 && '0' <= c && c <= '9' {
			continue
		}
		return false
	}

	return true
}
