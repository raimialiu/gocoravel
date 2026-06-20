package delegate

type Action[T any] func()
type Func[T interface{}, R any] func(payload ...T) R

type ActionOrAsyncFunc struct {
	_action      Action[interface{}]
	_asyncAction Func[interface{}, interface{}]
	_isAsync     bool
}
