package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEditConfigObject(t *testing.T) {
	testCases := []struct {
		baseline string
		action   string
		server   string
		helper   string
	}{
		{
			baseline: "none",
			action:   "added_credstore",
			helper:   "helper0",
		},
		{
			baseline: "none",
			action:   "added_helper",
			server:   "server1",
			helper:   "helper1",
		},
		{
			baseline: "credstore",
			action:   "changed",
			helper:   "helper0",
		},
		{
			baseline: "credstore",
			action:   "added_helper",
			server:   "server1",
			helper:   "helper1",
		},
		{
			baseline: "helpers_empty",
			action:   "added_helper",
			server:   "server1",
			helper:   "helper1",
		},
		{
			baseline: "helpers_existing",
			action:   "added_credstore",
			helper:   "helper0",
		},
		{
			baseline: "helpers_existing",
			action:   "added_helper",
			server:   "server2",
			helper:   "helper2",
		},
		{
			baseline: "helpers_existing",
			action:   "changed_helper",
			server:   "server1",
			helper:   "helper1",
		},
	}
	for iteration, tc := range testCases {
		fmt.Printf("Running iteration %d...\n", iteration)
		var err error
		var actual, expected *map[string]interface{}
		actual, err = loadConfigObject(fmt.Sprintf("./testcase/%s.config.json", tc.baseline))
		if err != nil {
			assert.Fail(t, fmt.Sprintf("ERROR: %s", err.Error()))
		}
		editConfigObject(actual, tc.server, tc.helper)
		expected, err = loadConfigObject(fmt.Sprintf("./testcase/%s.%s.config.json", tc.baseline, tc.action))
		if err != nil {
			assert.Fail(t, fmt.Sprintf("ERROR: %s", err.Error()))
		}
		var expectedStr, actualStr string
		expectedStr, err = toString(expected)
		if err != nil {
			assert.Fail(t, fmt.Sprintf("ERROR: %s", err.Error()))
		}
		actualStr, err = toString(actual)
		if err != nil {
			assert.Fail(t, fmt.Sprintf("ERROR: %s", err.Error()))
		}
		// well, we are counting on the order of properties being serialized being consistent
		assert.Equal(t, expectedStr, actualStr)
	}
}

func toString(obj *map[string]interface{}) (output string, err error) {
	var bytes []byte
	if bytes, err = json.MarshalIndent(obj, "\n", "\t"); err != nil {
		return "", err
	}
	return string(bytes), nil
}
