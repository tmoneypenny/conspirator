package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemovePortFromClientIP(t *testing.T) {
	assert.Equal(t, "1.1.1.1", RemovePortFromClientIP("1.1.1.1:45678"))
	assert.Equal(t, "1.1.1.1", RemovePortFromClientIP("1.1.1.1"))
	assert.Equal(t, "", RemovePortFromClientIP(""))
}
