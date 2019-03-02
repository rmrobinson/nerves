package policy

import (
	"sort"
	"sync"
)

// Engine contains a single instance of a policy engine.
// This engine contains one or more policies, subscribes to updates from one or more services
// to trigger conditional changes, and uses these subscribed services to execute one or more actions
// when the relevant policy executes.
type Engine struct {
	policies []*Policy
	policyLock sync.Mutex

	timers map[string]string

	currState State
}

// NewEngine creates a new policy engine.
func NewEngine() *Engine {
	return &Engine{
		policies: []*Policy{},
		timers: map[string]string{},
	}
}

// AddPolicy registers a new policy with the policy engine.
// Policies are held in an ordered list, descending by their weights, and this add will ensure
// the inserted policy is placed in the appropriate location.
func (e *Engine) AddPolicy(policy *Policy) {
	e.policyLock.Lock()
	defer e.policyLock.Unlock()

	e.policies = append(e.policies, policy)
	sort.Slice(e.policies, func(i, j int) bool {
		return e.policies[i].Weight < e.policies[j].Weight
	})
}
