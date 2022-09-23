package mtcp

type Head struct {
	PackLength int16
	Method     string
}

func (h *Head) Unmarshal([]byte) error {
	h.PackLength = 8
	h.Method = "1"
	return nil
}

func (h *Head) GetDataLength() int16 {
	return h.PackLength
}

func (h *Head) GetMethod() string {
	return h.Method
}
