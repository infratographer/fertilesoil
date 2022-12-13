package driver

// Options defines ways to configure the CRDB driver.
type Options func(*Driver)

// WithReadOnly configures the driver to be read-only.
func WithReadOnly() Options {
	return func(d *Driver) {
		d.readOnly = true
	}
}

// WithFastReads configures the driver to use fast reads.
// This is useful for read-only drivers. It will use a
// different query to read directories, which relies on
// follower reads. It is important to note that this
// query will not return the latest data, but it will
// return data that is consistent with the latest data.
func WithFastReads() Options {
	return func(d *Driver) {
		d.fastReads = true
	}
}
