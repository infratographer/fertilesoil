package storage

// GetOptions allows for additional options to be passed to get functions.
type GetOptions struct {
	WithDeleted bool
}

// IsWithDeleted returns WithDeleted.
// If opts is nil, false is returned.
func (opts *GetOptions) IsWithDeleted() bool {
	if opts == nil {
		return false
	}

	return opts.WithDeleted
}

// ListOptions allows for additional options to be passed to list functions.
type ListOptions struct {
	WithDeleted bool
}

// IsWithDeleted returns WithDeleted.
// If opts is nil, false is returned.
func (opts *ListOptions) IsWithDeleted() bool {
	if opts == nil {
		return false
	}

	return opts.WithDeleted
}

// Converts ListOptions to the equivalent GetOptions.
// Useful when a list call makes an underlying get call.
func (opts *ListOptions) ToGetOptions() *GetOptions {
	if opts == nil {
		return nil
	}

	return &GetOptions{
		WithDeleted: opts.WithDeleted,
	}
}
