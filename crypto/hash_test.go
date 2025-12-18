package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestHashBase64JavaExample(t *testing.T) {
	// Example from User
	// Input String: appId=5ca92f944739055d85c14ad32e65760a&channelNum=0&hospitalId=39199&requestId=BBBE1167DDE34F0DB6DE5A2620717132&timestamp=1766054396
	// Secret: fa44fa8e79e1d17a77387a6694cb9374
	// RawStr + Secret
	input := "appId=5ca92f944739055d85c14ad32e65760a&channelNum=0&hospitalId=39199&requestId=BBBE1167DDE34F0DB6DE5A2620717132&timestamp=1766054396fa44fa8e79e1d17a77387a6694cb9374"
	expected := "RiQtJO0oe9Ps9fP7Go1mUclqn15amjJ8POU3Z3vULag="

	// Test Go Function direct call
	res, err := Hash(HashTypes["SHA256"], input, "base64")
	assert.NoError(t, err)
	assert.Equal(t, expected, res)

	// Test Process call
	args := []interface{}{"SHA256", input, "base64"}
	resProcess := process.New("crypto.Hash", args...).Run()
	assert.Equal(t, expected, resProcess)
}
