package main

import (
	"account-switcher/internal/basic"

	_ "embed"
)

//go:embed GameStats.json
var embeddedGameStatsJSON []byte

func init() {
	basic.SetEmbeddedGameStatsJSON(embeddedGameStatsJSON)
}
