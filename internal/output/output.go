package output

import "fmt"

const (
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

func OK(format string, args ...any) {
	fmt.Printf(green+"[OK]"+reset+"   "+format+"\n", args...)
}

func Fail(format string, args ...any) {
	fmt.Printf(red+"[FAIL]"+reset+" "+format+"\n", args...)
}

func Info(format string, args ...any) {
	fmt.Printf(yellow+"[INFO]"+reset+" "+format+"\n", args...)
}