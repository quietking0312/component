package round_battle

type AddSource struct {
}

// Attribute 属性
type Attribute struct {
	base      int32      // 基础属性
	current   int32      // 当前属性
	actor     *Actor     // 属性所属对象
	addSource *AddSource // 加成来源
}

func (a Attribute) Value() int32 {
	return a.current
}

type PosAttribute struct {
	*Attribute
}

func (a *PosAttribute) Row() uint8 {
	if a.Value() == 10 {
		return 3
	} else if a.Value() > 6 {
		return 2
	} else {
		return 1
	}
}
