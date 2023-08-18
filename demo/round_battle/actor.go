package round_battle

type Actor struct {
	Id        uint32
	camp      int8
	Pos       *PosAttribute // 坐标
	HP        *Attribute
	MaxHP     *Attribute
	ATN       *Attribute
	INT       *Attribute
	DEF       *Attribute
	RES       *Attribute
	SPD       *Attribute //速度
	HIT       *Attribute
	Buffs     *Buffs
	DeBuffs   *Buffs
	BaseSkill *Skill  // 基础攻击手段
	Skills    *Skills // 技能
	State     uint32
}
