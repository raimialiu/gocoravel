package services

type (
	/*
		Imagine you hire a person whose only job is to sit in a room, stare at a clock,
		and every time the minute hand hits 12 (i.e. a new minute starts), they look at a list and ask
		"is anything due right now?". If yes, they run it. Then they go back to staring at the clock.
		That person,  that behaviour, is the behaviour of ClockWatcher or rather what it is meant to do.
	*/
	HostedService struct {
		_isCancellationRequested bool
		// need a loop that continue to run (in the background)
	} // watches the clock and determine if it is time to run a job
)
