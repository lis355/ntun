package ntun

type Listener interface {
	Listen() error
}

type Closer interface {
	Close() error
}
