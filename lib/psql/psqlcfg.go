package psql

import (
	. "../logx"
)

// psql configuration

type ConfigPSQL struct {
	ConnStr         string
	ConnMaxLifetime float64
	MaxIdleConns    int32
	MaxOpenConns    int32
	Logger          LoggerX
}

var DefaultConfigPSQL = ConfigPSQL{
	ConnStr:         "",
	ConnMaxLifetime: 0.0,
	MaxIdleConns:    0,
	MaxOpenConns:    0,
}
