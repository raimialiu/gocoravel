package delegate

import (
	"reflect"
	"runtime"
)

type ActionOrAsyncFuncConfig func(*ActionOrAsyncFunc)

func NewActionFunc(opts ...ActionOrAsyncFuncConfig) ActionOrAsyncFunc {
	options := &ActionOrAsyncFunc{}
	for _, opt := range opts {
		opt(options)
	}

	options._isAsync = options._asyncAction != nil
	return ActionOrAsyncFunc(*options)
}

func (as ActionOrAsyncFunc) Invoke() {
	if as._isAsync {
		go as._asyncAction()
	} else {
		as._action()
	}
}

// Run executes the action synchronously (even an async one) so a caller can time
// it and observe completion — used by the scheduler to record run history.
func (as ActionOrAsyncFunc) Run() {
	switch {
	case as._isAsync && as._asyncAction != nil:
		as._asyncAction()
	case as._action != nil:
		as._action()
	}
}

// FuncName returns the runtime name of the underlying function (e.g.
// "main.cleanup" or "main.main.func1"), or "" if there is none. It is stable for
// a given code location, so the scheduler uses it to derive a stable id for an
// unnamed closure-based job.
func (as ActionOrAsyncFunc) FuncName() string {
	var fn interface{}
	if as._isAsync {
		fn = as._asyncAction
	} else {
		fn = as._action
	}
	if fn == nil {
		return ""
	}
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return ""
	}
	pc := runtime.FuncForPC(v.Pointer())
	if pc == nil {
		return ""
	}
	return pc.Name()
}
