package database

import (
	"testing"

	"github.com/SOAT-15-Oficina/oficina-mecanica-monolith/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPoolConfigLimitsConnectionsPerReplica(t *testing.T) {
	tests := []struct {
		name             string
		configuredMax    int32
		expectedMaxConns int32
	}{
		{name: "safe default", configuredMax: 0, expectedMaxConns: 5},
		{name: "configured value", configuredMax: 3, expectedMaxConns: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolConfig, err := buildPoolConfig(&config.DatabaseConfig{
				User:           "user",
				Password:       "password",
				Host:           "postgres",
				Port:           "5432",
				Name:           "database",
				MaxConnections: tt.configuredMax,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMaxConns, poolConfig.MaxConns)
		})
	}
}
