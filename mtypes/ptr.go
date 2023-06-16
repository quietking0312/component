package mtypes

type T interface {
	string | int | int64 | int32 | int16 | int8 | uint | uint16 | uint32 | uint64 | uint8 | any
}

func Ptr[k T](s k) *k {
	return &s
}

func Value[k T](p *k) k {
	return *p
}
