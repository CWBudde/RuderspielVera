//go:build js && wasm

package main

import (
	"math/rand/v2"

	"ruderspiel/internal/bot"
	"ruderspiel/internal/engine"
	"ruderspiel/internal/gameservice"
)

// botDecider is the bot brain used in the browser; keep it in sync with cmd/server/bots.go.
var botDecider gameservice.Decider = func(g *engine.Game, playerID string, seat int, rng *rand.Rand) (engine.Decision, error) {
	return bot.Decide(g, playerID, bot.PersonalityFor(seat), rng)
}
