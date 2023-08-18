package round_battle

import "fmt"

const (
	BuffIdSeal = 1 // 封印
)

type BufferSourceStruct struct {
	ObjectData *Actor // buff 来源对象
	Skill      *Skill // buff来源技能
	CreateAt   int32  // buff 添加时间
}

type Buffs struct {
	buffs  map[int32]*Buff
	nextId int32
}

func NewBuffs() *Buffs {
	return &Buffs{
		buffs:  make(map[int32]*Buff),
		nextId: 10000,
	}
}

func (b *Buffs) Add(buff *Buff) error {
	if buff.Id >= 10000 {
		return fmt.Errorf("id > 10000")
	}
	if buff.Id == 0 {
		b.buffs[b.nextId] = buff
		b.nextId += 1
	} else {
		b.buffs[buff.Id] = buff
	}
	return nil
}

func (b *Buffs) Get(id int32) *Buff {
	return b.buffs[id]
}

func (b *Buffs) Range(fc func(buf *Buff)) {
	for _, buf := range b.buffs {
		fc(buf)
	}
}

type Buff struct {
	Id      int32  // buffId 相同的id 只存在一个
	Type    uint32 //buff 种类 表示buff效果
	MaxTime int32  // 最大持续时间
	Time    int32  // 剩余时间
	Number  uint16 // buff层数
	Source  *BufferSourceStruct
}

func NewBuff(Id int32, T uint32, maxTime int32, number uint16, sourceObject *Actor, skill *Skill, cAt int32) *Buff {
	return &Buff{
		Id:      Id,
		Type:    T,
		MaxTime: maxTime,
		Time:    maxTime,
		Number:  number,
		Source: &BufferSourceStruct{
			ObjectData: sourceObject,
			Skill:      skill,
			CreateAt:   cAt,
		},
	}
}
