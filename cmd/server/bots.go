package main

import "ruderspiel/internal/api"

// botDecider is the bot brain used by the server.
// TODO: switch to internal/bot (bot.Decide with bot.PersonalityFor(seat)) once it exists.
var botDecider api.Decider = api.SimpleDecider
