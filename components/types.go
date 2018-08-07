package components

type Component interface {
	Install() (Component, error)
	Namespace() string
	IsRunning() (bool, error)
}

type IngressComponent interface {
	ServiceName() string
	Namespace() string
}
