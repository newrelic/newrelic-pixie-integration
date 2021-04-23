package errors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigError(t *testing.T) {

	err := ConfigurationError("invalid value 'unknown' for property NewRelicRegion. Supported values are: eu, usa, fed, stg")
	assert.NotNil(t,err)
	assert.Equal(t, err.Error(),"[ERR] invalid value 'unknown' for property NewRelicRegion. Supported values are: eu, usa, fed, stg")
	assert.Implements(t, (*error)(nil), err)
}