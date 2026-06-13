package game

// damageDefs maps a damage kind to its base damage value. Adding a new damage
// kind is a single entry here (plus its damageKind* constant in game.go).
var damageDefs = map[string]int{
	damageKindFireball:  40,
	damageKindExplosion: 20,
	damageKindArrow:     30,
	damageKindSpike:     25,
	damageKindBullet:    25,
	damageKindFirespot:  20,
	damageKindLightning: 30,
}

const defaultDamage = 20

// damageForKind returns the base damage for a damage kind, or defaultDamage if
// the kind is unknown.
func damageForKind(kind string) int {
	if d, ok := damageDefs[kind]; ok {
		return d
	}

	return defaultDamage
}
