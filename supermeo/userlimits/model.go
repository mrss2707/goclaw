package userlimits

type UserQuota struct {
	MaxAgents        int `json:"max_agents"`
	MaxSessions      int `json:"max_sessions"`
	MaxTokensPerDay  int `json:"max_tokens_per_day"`
}
