package game

import "testing"

func TestDamageForKind(t *testing.T) {
	tests := []struct {
		kind string
		want int
	}{
		{damageKindFireball, 40},
		{damageKindExplosion, 20},
		{damageKindArrow, 30},
		{damageKindSpike, 25},
		{damageKindBullet, 25},
		{damageKindFirespot, 20},
		{damageKindLightning, 30},
		{"unknown_kind", defaultDamage},
		{"", defaultDamage},
	}
	for _, tc := range tests {
		if got := damageForKind(tc.kind); got != tc.want {
			t.Errorf("damageForKind(%q) = %d, want %d", tc.kind, got, tc.want)
		}
	}
}

func TestClassMaxHP(t *testing.T) {
	tests := []struct {
		class string
		want  int
	}{
		{ClassMage, 150},
		{ClassKnight, 250},
		{ClassRogue, 200},
		{"unknown", defaultClassMaxHP},
	}
	for _, tc := range tests {
		if got := classMaxHP(tc.class); got != tc.want {
			t.Errorf("classMaxHP(%q) = %d, want %d", tc.class, got, tc.want)
		}
	}
}

func TestClassResistance(t *testing.T) {
	tests := []struct {
		name  string
		class string
		kind  string
		want  float64
	}{
		{"mage resists fireball", ClassMage, damageKindFireball, 0.5},
		{"mage resists explosion", ClassMage, damageKindExplosion, 0.5},
		{"mage resists firespot", ClassMage, damageKindFirespot, 0.5},
		{"mage no resist to arrow", ClassMage, damageKindArrow, 1.0},
		{"knight resists spike", ClassKnight, damageKindSpike, 0.5},
		{"knight resists arrow", ClassKnight, damageKindArrow, 0.5},
		{"knight no resist to fireball", ClassKnight, damageKindFireball, 1.0},
		{"rogue resists bullet", ClassRogue, damageKindBullet, 0.5},
		{"rogue no resist to spike", ClassRogue, damageKindSpike, 1.0},
		{"unknown class no resist", "unknown", damageKindFireball, 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classResistance(tc.class, tc.kind); got != tc.want {
				t.Errorf("classResistance(%q, %q) = %v, want %v", tc.class, tc.kind, got, tc.want)
			}
		})
	}
}

func TestMonsterDefBySpawnName(t *testing.T) {
	if def := monsterDefBySpawnName("skeleton"); def == nil || def.Kind != monsterKindSkeleton {
		t.Errorf("expected skeleton def, got %v", def)
	}
	// Demon exists as a def but is also map-spawnable by name.
	if def := monsterDefBySpawnName("demon"); def == nil || def.Kind != monsterKindDemon {
		t.Errorf("expected demon def, got %v", def)
	}
	if def := monsterDefBySpawnName("nonexistent"); def != nil {
		t.Errorf("unknown spawn name should match nothing, got %v", def)
	}
}

func TestMonsterDefDefaultMoveSpeed(t *testing.T) {
	// Archer has no explicit MoveSpeed, so it should get the default.
	if def := monsterDefs[monsterKindArcher]; def.MoveSpeed != defaultMonsterMoveSpeed {
		t.Errorf("archer MoveSpeed = %d, want default %d", def.MoveSpeed, defaultMonsterMoveSpeed)
	}
	// Golem sets an explicit speed and should keep it.
	if def := monsterDefs[monsterKindGolem]; def.MoveSpeed != 1 {
		t.Errorf("golem MoveSpeed = %d, want 1", def.MoveSpeed)
	}
}
