package mnet

type Conn interface {
	Read() (int, []byte, error)
	Write(b []byte) (int, error)
	Close() error
}
