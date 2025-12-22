package logger

import (
	"fmt"
	"time"
)

func Println(msg ...any) {
	fmt.Printf(":: %s %s\n", time.Now().Format("15:04:05"), fmt.Sprint(msg...))
}

func Printf(format string, v ...interface{}) {
	fmt.Printf(":: %s %s\n",
		time.Now().Format("15:04:05"), fmt.Sprintf(format, v...))
}

func Shout(msg ...any) {
	fmt.Printf(":: %s === %s ===\n",
		time.Now().Format("15:04:05"), fmt.Sprint(msg...))
}

func Shoutf(format string, v ...interface{}) {
	fmt.Printf(":: %s === %s ===\n",
		time.Now().Format("15:04:05"), fmt.Sprintf(format, v...))
}
