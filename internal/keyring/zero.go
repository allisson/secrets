package keyring

// Zero overwrites every byte of b with zero, clearing sensitive key material.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
