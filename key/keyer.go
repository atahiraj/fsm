package key

// Keyer exposes a stable comparable key for a value.
type Keyer[K comparable] interface {
	Key() K
}
