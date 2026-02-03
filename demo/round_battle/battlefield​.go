package round_battle

type BattleField struct {
	members map[int8][]*Member
	axis    []*Member
	stat    *Stat
}

func NewBattleField() *BattleField {
	field := &BattleField{}
	return field
}

// Init 战场初始化
func (b *BattleField) Init() {

}

func (b *BattleField) Next() {

}
