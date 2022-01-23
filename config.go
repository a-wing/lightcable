package lightcable

// Config describes the configuration of the server.
// server should worker, every worker use this configuration
type Config struct {
	SignBufferCount int
	CastBufferCount int

	// If you set this option as `false`
	// The server will not broadcast to you messages you send.
	// Look like MQTTv5 nolocal
	Local bool

	Worker *Config
}

// DefaultConfig is a server with all fields set to the default values.
var DefaultConfig = &Config{
	SignBufferCount: 128,
	CastBufferCount: 128,
	Local:           false,

	Worker: &Config{
		SignBufferCount: 128,
		CastBufferCount: 128,
		Local:           false,
	},
}
