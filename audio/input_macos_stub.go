//go:build !darwin

package audio

import (
	"aprs_agent/config"
	"fmt"
)

// newMacOSInput stub for non-macOS systems
func newMacOSInput(cfg *config.Config, devices DeviceManagerInterface) (AudioInput, error) {
	return nil, fmt.Errorf("macOS audio input not available on this platform")
}
