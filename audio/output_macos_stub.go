//go:build !darwin

package audio

import (
	"aprs_agent/config"
	"fmt"
)

// newMacOSOutput stub for non-macOS systems
func newMacOSOutput(cfg *config.Config, devices DeviceManagerInterface) (AudioOutput, error) {
	return nil, fmt.Errorf("macOS audio output not available on this platform")
}
