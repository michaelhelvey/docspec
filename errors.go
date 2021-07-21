package docspec

import "fmt"

func errMsgPrefix() string {
	return "[docspec]:"
}

func invariantViolation(msg string) error {
	return fmt.Errorf("%s %s", errMsgPrefix(), msg)
}
