//go:build !windows

package catalogupdate

func EnsureDailyTask(string) error {
	return nil
}
