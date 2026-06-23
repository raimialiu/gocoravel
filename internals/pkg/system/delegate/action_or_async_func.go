package delegate

type ActionOrAsyncFunc struct {
	_action      Action[interface{}]
	_asyncAction Func[interface{}, interface{}]
	_isAsync     bool
}

func NewAction(action Action[interface{}], fun Func[interface{}, interface{}]) *ActionOrAsyncFunc {
	return &ActionOrAsyncFunc{
		_action:      action,
		_isAsync:     fun != nil,
		_asyncAction: fun,
	}
}
