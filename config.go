package lightcable

// Config describes the configuration of the server.
type Config struct {
	// register, unregister room buffer count
	SignBufferCount int
	// broadcast message to room buffer count
	CastBufferCount int

	Worker WorkerConfig
}

// Worker Config describes the configuration of the server.
// server should worker, every worker use this configuration
type WorkerConfig struct {
	// register, unregister client buffer count
	SignBufferCount int
	// broadcast message to client buffer count
	CastBufferCount int

	// If you set this option as `false`
	// The server will not broadcast to you messages you send.
	// Look like MQTTv5 nolocal
	Local bool
}

// DefaultConfig is a server with all fields set to the default values.
var DefaultConfig = &Config{
	SignBufferCount: 128,
	CastBufferCount: 128,

	Worker: WorkerConfig{
		SignBufferCount: 128,
		CastBufferCount: 128,
		Local:           false,
	},
}
