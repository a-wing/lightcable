package lightcable

type Config struct {
	SignBufferCount int
	CastBufferCount int

	Worker *Config
}

var DefaultConfig = &Config{
	SignBufferCount: 128,
	CastBufferCount: 128,

	Worker: &Config{
		SignBufferCount: 128,
		CastBufferCount: 128,
	},
}
