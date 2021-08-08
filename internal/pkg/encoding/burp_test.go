package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractInteraction(t *testing.T) {
	bm := &BurpMarshaller{Ndots: 2}
	testCases := []struct {
		input    string
		expected string
	}{
		{
			"src.properties",
			"src",
		},
		{
			"default.src.properties",
			"default",
		},
		{
			"143szqk330351pk7ff1ladihj8pl34nmrpg.default.src.properties",
			"143szqk330351pk7ff1ladihj8pl34nmrpg",
		},
		{
			"test.143szqk330351pk7ff1ladihj8pl34nmrpg.default.src.properties",
			"143szqk330351pk7ff1ladihj8pl34nmrpg",
		},
		{
			"extra.test.143szqk330351pk7ff1ladihj8pl34nmrpg.default.src.properties",
			"143szqk330351pk7ff1ladihj8pl34nmrpg",
		},
		{
			"extra.extra.test.143szqk330351pk7ff1ladihj8pl34nmrpg.default.src.properties",
			"143szqk330351pk7ff1ladihj8pl34nmrpg",
		},
	}

	for test := range testCases {
		assert.Equal(t, testCases[test].expected, bm.extractInteraction(testCases[test].input))
	}
}
