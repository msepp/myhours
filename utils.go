package myhours

// Val returns the value of any pointer, or the zero value of the type if pointer
// is nil.
func Val[T any](p *T) T {
	if p == nil {
		return *new(T)
	}
	return *p
}

// Ptr returns a pointer to a copy of given value.
func Ptr[T any](v T) *T {
	return &v
}

// PtrNonZero returns a pointer a copy of given value or nil if value is the
// zero value of the type.
func PtrNonZero[T comparable](p T) *T {
	if *new(T) == p {
		return nil
	}
	return &p
}
