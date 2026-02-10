package domain

// Zero securely overwrites a byte slice with zeros to clear sensitive data from memory.
func Zero(b []byte) {
	if b == nil {
		return
	}
	for i := range b {
		b[i] = 0
	}
}
