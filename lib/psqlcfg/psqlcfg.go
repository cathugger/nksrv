package psqlcfg

// psql configuration

type ConfigPSQL struct {
	ConnStr         string  `toml:"connect_string"`
	ConnMaxLifetime float64 `toml:"connection_max_lifetime"`
	MaxIdleConns    int32   `toml:"max_idle_connections"`
	MaxOpenConns    int32   `toml:"max_open_connections"`
}

var DefaultConfigPSQL = ConfigPSQL{
	ConnStr:         "",
	ConnMaxLifetime: 0.0,
	MaxIdleConns:    0,
	MaxOpenConns:    0,
}
