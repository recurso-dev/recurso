package domain

import "time"

// BanditArm represents a time slot for retrying payments
type BanditArm string

const (
	ArmMorning   BanditArm = "morning"   // 06:00 - 12:00
	ArmAfternoon BanditArm = "afternoon" // 12:00 - 18:00
	ArmEvening   BanditArm = "evening"   // 18:00 - 00:00
	ArmNight     BanditArm = "night"     // 00:00 - 06:00
)

// BanditStats tracks the success rate of each arm
type BanditStats struct {
	Arm         BanditArm `json:"arm"`
	Successes   int       `json:"successes"`
	Failures    int       `json:"failures"`
	LastUpdated time.Time `json:"last_updated"`
}

// GetSuccessRate returns the success rate (0.0 - 1.0)
func (b *BanditStats) GetSuccessRate() float64 {
	total := b.Successes + b.Failures
	if total == 0 {
		return 0.0
	}
	return float64(b.Successes) / float64(total)
}
