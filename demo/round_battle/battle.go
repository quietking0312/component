package round_battle

import (
	"math/rand"
	"sort"
	"time"
)

type P struct {
	Actors []*Actor
	Buff   []*Buff
	camp   int
}

type Battle struct {
	Battlefield [2]*P
	Line        []*Actor
	NewLine     []*Actor
	R           *rand.Rand
}

func NewBattle(one *P, two *P) *Battle {
	battle := &Battle{
		R: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	one.camp = 0
	battle.Battlefield[0] = one
	two.camp = 1
	battle.Battlefield[1] = two

	battle.Line = append(battle.Line, one.Actors...)
	battle.Line = append(battle.Line, two.Actors...)
	sort.Slice(battle.Line, func(i, j int) bool {
		v1 := battle.Line[i]
		v2 := battle.Line[j]
		if v1.Pos.Row() != v2.Pos.Row() { // 后排先出手
			return v1.Pos.Row() > v2.Pos.Row()
		}
		if v1.SPD.Value() != v2.SPD.Value() { // 速度快的先出手
			return v1.SPD.Value() > v2.SPD.Value()
		}
		return battle.R.Intn(100) < 50 // 随机出手
	})
	battle.NewLine = battle.Line
	return battle
}

func (b *Battle) Start() {
	round := 1
	for {
		// 计算增益buff
		b.Line = b.NewLine
		// 开始出手
		for _, actor := range b.Line {
			if actor.HP.Value() <= 0 {
				continue
			}
			sealBuff := actor.DeBuffs.Get(BuffIdSeal)
			if sealBuff != nil && sealBuff.Number > 0 {
				sealBuff.Number -= 1
				continue
			}
			comNum := 1
			// 计算连击次数

			for c := 1; c <= comNum; c++ {
				actor.BaseSkill.Execute(b.Battlefield)
			}

		}

		round += 1
	}
}
