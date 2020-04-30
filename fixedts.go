package fixedts

import (
	"context"
	"sync"
	"time"
)

// TODO: Improve synchronization pattern. Acceptable error.

// If passed a timestep smaller than a nanosecond, a nanosecond will be used
const MinimumTimestep = 0.000000001

// If a non-positive acceptable error value is passed, 0.1% of the passed timestep is used
const DefaultAcceptableErrorRate = 0.001

// FixedTimeStepFunc is a function to execute at a fixed timesstep.
type FixedTimestepFunc func(*FixedTimestep)

// FixedTimestep executes a FixedTimestepFunc as a callback at the desired timestep.
// Reading the fields of the struct is safe only from the callback.
type FixedTimestep struct {
	// StepsUntilPause is the remaining steps until automatically pausing
	StepsUntilPause uint64
	// Now is the time when this step started
	Now time.Time
	// Step is the currently executing step
	Step uint64
	// Delta is the elapsed since the last step.
	// It's zero on the first step or when starting to play after a pause.
	Delta float64
	// Gap is the deviation from the desired timestep from the last step.
	// It's zero on the first step or when starting to play after a pause.
	Gap float64
	// TotalStats gives statistics from the moment the timestep was first started.
	TotalStats *Stats
	// CurrentStats gives statistics from the moment the timestep was last started.
	CurrentStats *Stats
	// Config defines the behaviour of the timestep
	Config       *FixedTimestepConfig
	function     FixedTimestepFunc
	ticker       *time.Ticker
	lastStepTime time.Time
	controlCh    chan *FixedTimestepConfig
}

// FixedTimestepConfig describes the desired behaviour. Can be passed when calling
// `NewWithConfig` or while running calling `Update`.
type FixedTimestepConfig struct {
	// Paused pauses the execution of your function.
	// When started, it will call your function at de desired timestep (for each step
	// it first calls your function and then waits if any time left.
	// By default it is set to false, so the timestep is playing when created.
	Paused bool
	// DesiredTimestep sets desired timestep.
	// It's set to `MinimumTimestep` if smaller.
	DesiredTimestep float64
	// AcceptableError makes statistics to not count a step as delayed if within
	// the specified error respect to the step delta.
	// It's set to `DefaultAcceptableErrorRate * FixedTimestepConfig.DesiredTimestep` if
	// not positive.
	AcceptableError float64
	// StepsUntilPause can be set to a number higher than 0 to make the timestep
	// pause itself after having executed the specified step number.
	// The default value, 0, makes it play without pausing itself.
	StepsUntilPause uint64
	// Context can be optionally passed, so closing it causes the timestep to stop
	// and free its resources.
	Context context.Context
	// WaitGroup can be optionally passed, so the timestep will call `Done` on it when
	// exiting.
	WaitGroup *sync.WaitGroup
}

// New creates a timestep with the default config (see `FixedTimestepConfig`).
func New(desiredTimestep float64, function FixedTimestepFunc) *FixedTimestep {
	return NewWithConfig(&FixedTimestepConfig{
		DesiredTimestep: desiredTimestep,
	}, function)
}

// NewWithConfig creates a timestep with the passed config (see `FixedTimestepConfig`).
func NewWithConfig(config *FixedTimestepConfig, function FixedTimestepFunc) *FixedTimestep {
	config.defaults()
	ts := &FixedTimestep{
		StepsUntilPause: config.StepsUntilPause,
		Config:          config,
		function:        function,
		controlCh:       make(chan *FixedTimestepConfig),
	}
	ts.TotalStats = ts.newStats()
	go ts.run()
	return ts
}

func (config *FixedTimestepConfig) defaults() {
	if config.Context == nil {
		config.Context = context.Background()
	}
	if config.DesiredTimestep < MinimumTimestep {
		config.DesiredTimestep = MinimumTimestep
	}
	if config.AcceptableError <= 0 {
		config.AcceptableError = DefaultAcceptableErrorRate * config.DesiredTimestep
	}
}

// Update sets a new config with the desired behaviour for the timestep.
// It is safe to use concurrently from multiple goroutines.
// Calling `Update()` from multiple goroutines or from the callback is safe. From the
// callback it won't block, from outside it will block until the defined `FixedTimestepFunc`
// callback is exectued.
// It allows, to start, stop, change the desired time step... (see `FixedTimestepConfig`).
func (ts *FixedTimestep) Update(config *FixedTimestepConfig) {
	ts.controlCh <- config
}

func (ts *FixedTimestep) run() {
RunLoop:
	for {
		// Enter paused state (wait exit or reconfig)
		if ts.Config.Paused {
			select {
			case <-ts.Config.Context.Done(): // exit
				break RunLoop
			case config := <-ts.controlCh: // reconfig
				config.defaults()
				ts.Config = config
				continue RunLoop
			}
		}

		// Enter playing state
		ts.lastStepTime = time.Time{}
		ts.StepsUntilPause = ts.Config.StepsUntilPause
		ts.CurrentStats = ts.newStats()
		ts.ticker = time.NewTicker(time.Duration(ts.Config.DesiredTimestep * float64(time.Second)))

		for {
			ts.Now = time.Now()
			if !ts.lastStepTime.IsZero() {
				ts.Delta = ts.Now.Sub(ts.lastStepTime).Seconds()
				ts.Gap = ts.Delta
				ts.updateStats(ts.TotalStats)
				ts.updateStats(ts.CurrentStats)
			}
			ts.function(ts)
			ts.Step++
			ts.StepsUntilPause--
			ts.lastStepTime = ts.Now

			if ts.Config.StepsUntilPause != 0 && ts.StepsUntilPause == 0 { // pause after specified steps
				ts.Config.Paused = true
				ts.ticker.Stop()
				continue RunLoop
			}

			select {
			case <-ts.Config.Context.Done(): // exit
				ts.ticker.Stop()
				break RunLoop
			case config := <-ts.controlCh: // reconfig
				config.defaults()
				ts.Config = config
				ts.ticker.Stop()
				continue RunLoop
			case <-ts.ticker.C: // tick
			}
		}
	}

	if ts.Config.WaitGroup != nil {
		ts.Config.WaitGroup.Done()
	}
}
