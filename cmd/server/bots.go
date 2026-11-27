package main

import (
	"math/rand/v2"

	"ruderspiel/internal/api"
	"ruderspiel/internal/bot"
	"ruderspiel/internal/engine"
)

// botDecider is the bot brain used by the server: odd seats play Taktiker, even seats Bauchgefühl.
var botDecider api.Decider = func(g *engine.Game, playerID string, seat int, rng *rand.Rand) (engine.Decision, error) {
	return bot.Decide(g, playerID, bot.PersonalityFor(seat), rng)
}
