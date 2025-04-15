package myhours

// Val returns the value of any pointer, or the zero value of the type if pointer
// is nil.
func Val[T any](p *T) T {
	if p == nil {
		return *new(T)
	}
	return *p
}
