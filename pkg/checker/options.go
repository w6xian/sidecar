package checker

type Option func(*Options)

type Options struct {
	Min          float64
	Max          float64
	defaultValue any
	Required     bool
	checker      func(v any) (any, error)
}

func newOptions(opt ...Option) Options {
	def := Options{
		Min:          0,
		Max:          0,
		defaultValue: nil,
		Required:     true,
	}
	for _, o := range opt {
		o(&def)
	}

	return def
}

func Min(min float64) Option {
	return func(o *Options) {
		o.Min = min
	}
}
func Max(max float64) Option {
	return func(o *Options) {
		o.Max = max
	}
}

func Required(req bool) Option {
	return func(o *Options) {
		o.Required = req
	}
}

func DefaultValue(def any) Option {
	return func(o *Options) {
		o.defaultValue = def
	}
}

func Func(ck CheckFunc) Option {
	return func(o *Options) {
		o.checker = ck
	}
}

func String(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_STRING,
	}
	c.opts = newOptions(option...)
	return c
}

func Strings(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_STRINGS,
	}
	c.opts = newOptions(option...)
	return c
}
func Int(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_INT,
	}
	c.opts = newOptions(option...)
	return c
}

func Int32(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_INT32,
	}
	c.opts = newOptions(option...)
	return c
}
func Int64(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_INT64,
	}
	c.opts = newOptions(option...)
	return c
}

// uint
func Uint(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_UINT,
	}
	c.opts = newOptions(option...)
	return c
}

// uint32
func Uint32(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_UINT32,
	}
	c.opts = newOptions(option...)
	return c
}

// uint64
func Uint64(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_UINT64,
	}
	c.opts = newOptions(option...)
	return c
}

// float32
func Float32(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_FLOAT32,
	}
	c.opts = newOptions(option...)
	return c
}

// float64
func Float64(name string, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_FLOAT64,
	}
	c.opts = newOptions(option...)
	return c
}

type CheckFunc func(v any) (any, error)

// func
func Others(name string, cf CheckFunc, option ...Option) *Checker {
	c := &Checker{
		Name: name,
		Type: CHECKER_TYPE_OTHER,
	}
	opt := Func(cf)
	c.opts = newOptions(append([]Option{opt}, option...)...)
	return c
}
