package lightcable

// Server need config
type Config struct {
	SignBufferCount int
	CastBufferCount int

	Worker *Config
}

// Default Server config
var DefaultConfig = &Config{
	SignBufferCount: 128,
	CastBufferCount: 128,

	Worker: &Config{
		SignBufferCount: 128,
		CastBufferCount: 128,
	},
}
