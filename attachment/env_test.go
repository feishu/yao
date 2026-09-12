package attachment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceEnvFallback(t *testing.T) {
	// Case 1: Pure fallback - only S3 env is set
	os.Setenv("S3_API", "http://192.168.1.100:9000")
	os.Setenv("S3_ACCESS_KEY", "s3-test-key")
	os.Unsetenv("OBS_API")
	os.Unsetenv("OBS_ACCESS_KEY")
	defer func() {
		os.Unsetenv("S3_API")
		os.Unsetenv("S3_ACCESS_KEY")
	}()

	opt := &ManagerOption{
		Options: map[string]interface{}{
			"endpoint": "$ENV.OBS_API || $ENV.S3_API",
			"key":      "$ENV.OBS_ACCESS_KEY || $ENV.S3_ACCESS_KEY",
			"region":   "$ENV.OBS_REGION || $ENV.S3_REGION || cn-north-4",
		},
	}

	opt.ReplaceEnv("/test/root")

	assert.Equal(t, "http://192.168.1.100:9000", opt.Options["endpoint"])
	assert.Equal(t, "s3-test-key", opt.Options["key"])
	assert.Equal(t, "cn-north-4", opt.Options["region"]) // Literal fallback

	// Case 2: OBS env is set - takes precedence over S3 env
	os.Setenv("OBS_API", "https://obs.cn-east-3.myhuaweicloud.com")
	os.Setenv("OBS_ACCESS_KEY", "obs-specific-key")
	os.Setenv("OBS_REGION", "cn-east-3")
	defer func() {
		os.Unsetenv("OBS_API")
		os.Unsetenv("OBS_ACCESS_KEY")
		os.Unsetenv("OBS_REGION")
	}()

	opt2 := &ManagerOption{
		Options: map[string]interface{}{
			"endpoint": "$ENV.OBS_API || $ENV.S3_API",
			"key":      "$ENV.OBS_ACCESS_KEY || $ENV.S3_ACCESS_KEY",
			"region":   "$ENV.OBS_REGION || $ENV.S3_REGION || cn-north-4",
		},
	}

	opt2.ReplaceEnv("/test/root")

	assert.Equal(t, "https://obs.cn-east-3.myhuaweicloud.com", opt2.Options["endpoint"])
	assert.Equal(t, "obs-specific-key", opt2.Options["key"])
	assert.Equal(t, "cn-east-3", opt2.Options["region"])

	// Case 3: Backward compatibility with single $ENV. variable
	opt3 := &ManagerOption{
		Options: map[string]interface{}{
			"endpoint": "$ENV.OBS_API",
			"missing":  "$ENV.NON_EXISTENT_ENV_VAR",
		},
	}
	opt3.ReplaceEnv("/test/root")
	assert.Equal(t, "https://obs.cn-east-3.myhuaweicloud.com", opt3.Options["endpoint"])
	assert.Equal(t, "", opt3.Options["missing"])
}
