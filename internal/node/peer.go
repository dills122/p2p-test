package node

import "time"

type Peer struct {
	Addr          string
	Status        string
	LastSeen      time.Time
	Score         int
	Failures      int
	LastRTT       time.Duration
	CooldownUntil time.Time
}
