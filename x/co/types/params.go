package types

func NewParams() Params {
	return Params{}
}

func DefaultParams() Params {
	return NewParams()
}

// Validate returns nil; the module has no params today.
func (p Params) Validate() error { return nil }
