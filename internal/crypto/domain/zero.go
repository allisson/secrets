package domain

// zero securely overwrites a byte slice with zeros to remove sensitive data from memory.
//
// This function is used to clear sensitive cryptographic material (like keys) from
// memory after they are no longer needed. While Go's garbage collector will eventually
// reclaim the memory, explicitly zeroing ensures the data doesn't remain in memory
// longer than necessary.
//
// Security note: This provides defense in depth but is not a complete solution against
// sophisticated attacks (e.g., memory dumps, swap files). For maximum security, consider
// using memory locking (mlock) or dedicated secure memory management libraries.
//
// Parameters:
//   - b: The byte slice to zero (no-op if nil)
func zero(b []byte) {
	if b == nil {
		return
	}
	for i := range b {
		b[i] = 0
	}
}
