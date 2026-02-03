package round_battle

type Member struct {
	HP     int64
	AttAck int64
	isDie  bool // 是否死亡
}

// Action 行动
func (m *Member) Action() {

}

// TakeDamage 承受伤害
func (m *Member) TakeDamage(damage int64) {

}

func (m *Member) IsDie() bool {
	return m.isDie
}
