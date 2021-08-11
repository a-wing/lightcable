package lightcable

// Config describes the configuration of the server.
// server should worker, every worker use this configuration
type Config struct {
	SignBufferCount int
	CastBufferCount int

	Worker *Config
}

// DefaultConfig is a server with all fields set to the default values.
var DefaultConfig = &Config{
	SignBufferCount: 128,
	CastBufferCount: 128,

	Worker: &Config{
		SignBufferCount: 128,
		CastBufferCount: 128,
	},
}
