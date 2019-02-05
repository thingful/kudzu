package simular

import (
	"os"
)

const envVarName = "GONOMOCKS"

// Disabled returns true if the GONOMOCKS environment variable is not empty
func Disabled() bool {
	return os.Getenv(envVarName) != ""
}
