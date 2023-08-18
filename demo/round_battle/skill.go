package round_battle

type Skills struct {
	skills map[int32]*Skill
}

type Skill struct {
	Id    int32 // 技能id
	Owner *Actor
}

// Execute 执行技能
func (s *Skill) Execute(Battlefield [2]*P) {

	switch s.Id {

	default: // 平a 一次

	}
}

func (s *Skill) getTargets() {

}
