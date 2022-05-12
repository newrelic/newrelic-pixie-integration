package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetEndpoint(t *testing.T) {
	assert.Equal(t, "different.endpoint", getEndpoint("different.endpoint", "anything"))
	assert.Equal(t, endpointUSA, getEndpoint("", "anything"))
	assert.Equal(t, endpointEU, getEndpoint("", "eu01-xxxx"))
}
