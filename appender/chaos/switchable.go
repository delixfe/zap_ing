package chaos

type Switchable interface {
	Enabled() bool
	Enable()
	Disable()
}
