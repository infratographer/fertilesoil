package storage

// Options contains all possible storage options.
type Options struct {
	WithDeletedDirectories bool
}

// Option defines a storage Options handler.
type Option func(opts *Options)

var WithDeletedDirectories Option = func(opts *Options) {
	opts.WithDeletedDirectories = true
}

// BuildOptions applies all options to a new Options instance.
func BuildOptions(opts []Option) *Options {
	options := new(Options)

	for _, opt := range opts {
		opt(options)
	}

	return options
}
