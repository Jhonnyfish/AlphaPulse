package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCN(t *testing.T) {
	// Test with hours and minutes
	result := formatCN("%d小时%d分钟", 2, 30)
	assert.Equal(t, "2小时30分钟", result)

	// Test with minutes only
	result = formatCN("%d分钟", 45)
	assert.Equal(t, "45分钟", result)

	// Test with zero
	result = formatCN("%d分钟", 0)
	assert.Equal(t, "0分钟", result)

	// Test with no format specifiers
	result = formatCN("no numbers here")
	assert.Equal(t, "no numbers here", result)

	// Test with large numbers
	result = formatCN("%d小时", 100)
	assert.Equal(t, "100小时", result)
}

func TestAppendInt(t *testing.T) {
	// Test zero
	buf := make([]byte, 0, 10)
	result := appendInt(buf, 0)
	assert.Equal(t, "0", string(result))

	// Test positive number
	buf = make([]byte, 0, 10)
	result = appendInt(buf, 42)
	assert.Equal(t, "42", string(result))

	// Test negative number
	buf = make([]byte, 0, 10)
	result = appendInt(buf, -5)
	assert.Equal(t, "-5", string(result))

	// Test large number
	buf = make([]byte, 0, 10)
	result = appendInt(buf, 99999)
	assert.Equal(t, "99999", string(result))

	// Test single digit
	buf = make([]byte, 0, 10)
	result = appendInt(buf, 7)
	assert.Equal(t, "7", string(result))

	// Test appending to existing buffer
	buf = []byte("prefix:")
	result = appendInt(buf, 123)
	assert.Equal(t, "prefix:123", string(result))
}
