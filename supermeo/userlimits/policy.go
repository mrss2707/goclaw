package userlimits

import (
	"fmt"
)

var DefaultQuota = UserQuota{
	MaxAgents:       10,
	MaxSessions:     100,
	MaxTokensPerDay: 1000000,
}

func CheckCanCreateAgent(currentCount int, quota UserQuota) error {
	if quota.MaxAgents > 0 && currentCount >= quota.MaxAgents {
		return fmt.Errorf("agent limit exceeded: max %d", quota.MaxAgents)
	}
	return nil
}

func CheckCanCreateSession(currentCount int, quota UserQuota) error {
	if quota.MaxSessions > 0 && currentCount >= quota.MaxSessions {
		return fmt.Errorf("session limit exceeded: max %d", quota.MaxSessions)
	}
	return nil
}
