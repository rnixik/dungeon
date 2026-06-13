package game

// ClassDef holds the static configuration for a player class. Adding a new class
// is a single entry in classDefs (plus its Class* constant in game.go and an
// entry in classList).
type ClassDef struct {
	Name        string
	MaxHP       int
	Resistances map[string]float64 // damage kind -> damage multiplier (e.g. 0.5 = half)
}

// classList is the set of classes a random player can be assigned, in a stable
// order.
var classList = []string{ClassMage, ClassKnight, ClassRogue}

var classDefs = map[string]*ClassDef{
	ClassMage: {
		Name:  ClassMage,
		MaxHP: 150,
		Resistances: map[string]float64{
			damageKindFireball:  0.5,
			damageKindExplosion: 0.5,
			damageKindFirespot:  0.5,
		},
	},
	ClassKnight: {
		Name:  ClassKnight,
		MaxHP: 250,
		Resistances: map[string]float64{
			damageKindSpike: 0.5,
			damageKindArrow: 0.5,
		},
	},
	ClassRogue: {
		Name:  ClassRogue,
		MaxHP: 200,
		Resistances: map[string]float64{
			damageKindBullet: 0.5,
		},
	},
}

const defaultClassMaxHP = 100

// classMaxHP returns the max HP for a class, or defaultClassMaxHP if unknown.
func classMaxHP(class string) int {
	if def, ok := classDefs[class]; ok {
		return def.MaxHP
	}

	return defaultClassMaxHP
}

// classResistance returns the damage multiplier a class has against a damage
// kind, or 1.0 (no resistance) if none is defined.
func classResistance(class, kind string) float64 {
	if def, ok := classDefs[class]; ok {
		if mult, ok := def.Resistances[kind]; ok {
			return mult
		}
	}

	return 1.0
}
