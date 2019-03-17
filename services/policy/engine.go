package policy

import (
	"context"
	"sort"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/rmrobinson/nerves/services/domotics"
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

	if policy.Condition == nil || !policy.Condition.validate() {
		e.logger.Info("error validating policy, not adding",
			zap.String("name", policy.Name),
		)
		return
	}

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

	// Force a re-evaluation since we have a policy whose state may match.
	go func() {
		e.refresh <- true
	}()
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
		e.executePolicy(ctx, policy)
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

func (e *Engine) executePolicy(ctx context.Context, p *Policy) {
	if !p.Condition.triggered(e.state) {
		e.logger.Debug("policy conditions not met",
			zap.String("name", p.Name),
		)
		return
	}

	e.logger.Debug("policy conditions met, executing actions")
	for _, action := range p.Actions {
		e.executeAction(ctx, action)
	}
}

func (e *Engine) executeAction(ctx context.Context, a *Action) {
	switch a.Type {
	case Action_LOG:
		e.logger.Info("executing action",
			zap.String("name", a.Name),
		)
	case Action_DEVICE:
		e.logger.Debug("received device action",
			zap.String("name", a.Name),
		)

		deviceAction := &DeviceAction{}
		err := ptypes.UnmarshalAny(a.Details, deviceAction)
		if err != nil {
			e.logger.Info("error unmarshaling details",
				zap.String("name", a.Name),
				zap.Error(err),
			)
			return
		}

		if device, ok := e.state.deviceState[deviceAction.Id]; ok {
			proto.Merge(device.State, deviceAction.State)

			// We don't save the result as the monitor channel will pick up the update when it is broadcast.
			_, err := e.state.deviceClient.SetDeviceState(ctx, &domotics.SetDeviceStateRequest{
				Id:    deviceAction.Id,
				State: device.State,
			})
			if err != nil {
				e.logger.Info("error setting device state",
					zap.String("name", a.Name),
					zap.Error(err),
				)
			}
		} else {
			e.logger.Info("action with missing device id",
				zap.String("name", a.Name),
				zap.String("device_id", deviceAction.Id),
			)
		}
	}
}
