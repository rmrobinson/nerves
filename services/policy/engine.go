package policy

import (
	"context"
	"sort"
	"sync"

	"go.uber.org/zap"
)

// Engine contains a single instance of a policy engine.
// This engine contains one or more policies, subscribes to updates from one or more services
// to trigger conditional changes, and uses these subscribed services to execute one or more actions
// when the relevant policy executes.
type Engine struct {
	logger *zap.Logger

	refresh chan bool
	done    chan bool

	policies   []*Policy
	policyLock sync.Mutex

	timers map[string]string

	state *State
}

// NewEngine creates a new policy engine.
func NewEngine(logger *zap.Logger, state *State) *Engine {
	return &Engine{
		logger:   logger,
		refresh:  make(chan bool, 8),
		done:     make(chan bool),
		policies: []*Policy{},
		timers:   map[string]string{},
		state:    state,
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

// Refresh returns a channel that can be written to to trigger a new policy execution.
func (e *Engine) Refresh() chan<- bool {
	return e.refresh
}

// Done allows observers to watch for when the policy execution has completed.
func (e *Engine) Done() <-chan bool {
	return e.done
}

// Run begins the policy evaluation process.
// The evaluation can be terminated by aborting the supplied context.
func (e *Engine) Run(ctx context.Context) {
	for {
		select {
		case _ = <-ctx.Done():
			e.logger.Info("context cancelled, run done")
			go func() {
				e.done <- true
			}()
			return
		case refresh := <-e.refresh:
			if !refresh {
				continue
			}

			e.execute(ctx)
		}
	}
}

func (e *Engine) execute(ctx context.Context) {
	for _, policy := range e.policies {
		policy.execute(e.state)
	}
}
