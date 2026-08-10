//go:build windows

package catalogupdate

import (
	"fmt"
	"os/exec"
	"strings"
)

const TaskName = "Bumblebee Catalog Check"

func EnsureDailyTask(exePath string) error {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return fmt.Errorf("empty executable path")
	}
	tr := fmt.Sprintf(`"%s" --check-catalog`, exePath)
	cmd := exec.Command("schtasks", "/Create", "/TN", TaskName, "/TR", tr, "/SC", "DAILY", "/ST", "09:00", "/F", "/RL", "LIMITED")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
