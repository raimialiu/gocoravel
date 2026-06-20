package delegate

type ActionOrAsyncFuncConfig func(*ActionOrAsyncFunc)

func NewActionFunc(opts ...ActionOrAsyncFuncConfig) ActionOrAsyncFunc {
	options := &ActionOrAsyncFunc{}
	for _, opt := range opts {
		opt(options)
	}

	options._isAsync = options._asyncAction != nil
	return ActionOrAsyncFunc(*options)
}

func (as ActionOrAsyncFunc) Invoke() <-chan string {
	if as._isAsync {
		go as._asyncAction()
	} else {
		as._action()
	}

	result := make(chan string)
	result <- "done"
	return result
}
