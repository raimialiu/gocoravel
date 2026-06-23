package delegate

type Action[T any] func()
type Func[T interface{}, R any] func(payload ...T) R
