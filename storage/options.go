package storage

const (
	// DefaultPageSize is the default count of items returned in a list when no limit is provided.
	DefaultPageSize = 10
)

// Options contains all possible storage options.
type Options struct {
	// WithDeletedDirectories includes deleted directories in queries.
	WithDeletedDirectories bool

	// Page sets the page offset.
	Page int

	// PageSize sets the limit per page.
	PageSize int
}

// GetPage returns the page if defined.
// If Page is 0, the response is 1.
func (o *Options) GetPage() int {
	if o.Page < 1 {
		return 1
	}

	return o.Page
}

// GetPageSize returns the page size if defined.
// If PageSize is less than 1, the default page size of 10 is used.
func (o *Options) GetPageSize() int {
	if o.PageSize < 1 {
		return DefaultPageSize
	}

	return o.PageSize
}

// GetPageOffset returns the offset calculated by PageSize * (Page - 1).
func (o *Options) GetPageOffset() int {
	return o.GetPageSize() * (o.GetPage() - 1)
}

// Option defines a storage Options handler.
type Option func(opts *Options)

// WithDeletedDirectories includes deleted directories in queries.
var WithDeletedDirectories Option = func(opts *Options) {
	opts.WithDeletedDirectories = true
}

// Pagination sets the page and page size to be fetched.
func Pagination(page, size int) Option {
	return func(opts *Options) {
		opts.Page = page
		opts.PageSize = size
	}
}

// BuildOptions applies all options to a new Options instance.
func BuildOptions(opts []Option) *Options {
	options := new(Options)

	for _, opt := range opts {
		opt(options)
	}

	return options
}
