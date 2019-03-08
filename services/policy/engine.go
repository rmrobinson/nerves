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

	state *State
}

// NewEngine creates a new policy engine.
func NewEngine(logger *zap.Logger, state *State) *Engine {
	engine := &Engine{
		logger:   logger,
		refresh:  make(chan bool, 8),
		done:     make(chan bool),
		policies: []*Policy{},
		state:    state,
	}

	state.refresh = engine.refresh
	return engine
}

// AddPolicy registers a new policy with the policy engine.
// Policies are held in an ordered list, descending by their weights, and this add will ensure
// the inserted policy is placed in the appropriate location.
func (e *Engine) AddPolicy(policy *Policy) {
	e.policyLock.Lock()
	defer e.policyLock.Unlock()

	if !e.setupPolicy(policy) {
		e.logger.Info("error setting up policy, not adding",
			zap.String("name", policy.Name),
		)
		return
	}

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
	e.logger.Debug("starting event loop")

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

			e.logger.Debug("refresh triggered")
			e.execute(ctx)
		}
	}
}

func (e *Engine) setupPolicy(policy *Policy) bool {
	cronConditions := findCronConditions(policy.Condition)

	for _, cronCondition := range cronConditions {
		err := e.state.addCronEntry(cronCondition)
		if err != nil {
			e.logger.Info("error adding cron condition",
				zap.String("name", policy.Name),
				zap.Error(err),
			)
			return false
		}
	}

	return true
}

func (e *Engine) execute(ctx context.Context) {
	for _, policy := range e.policies {
		policy.execute(e.logger, e.state)
	}
}

func findCronConditions(c *Condition) []*Condition {
	if c.Cron != nil {
		return []*Condition{c}
	} else if c.Set == nil {
		return nil
	}

	var ret []*Condition
	for _, cond := range c.Set.Conditions {
		ret = append(ret, findCronConditions(cond)...)
	}

	return ret
}
