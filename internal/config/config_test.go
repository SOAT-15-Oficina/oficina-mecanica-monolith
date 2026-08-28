package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Server)
	assert.NotNil(t, cfg.Database)
	assert.Equal(t, int32(5), cfg.Database.MaxConnections)
	assert.NotNil(t, cfg.JWT)
	assert.NotNil(t, cfg.Email)
}
