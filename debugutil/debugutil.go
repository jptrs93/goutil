package debugutil

import (
	"fmt"
	"runtime"
	"strings"
)

func PrintStackInfo() {

	buf := make([]byte, 1<<16)
	stackLen := runtime.Stack(buf, true)
	stackStr := string(buf[:stackLen])

	goroutines := strings.Split(stackStr, "\n\n")
	for _, goroutine := range goroutines {
		if strings.TrimSpace(goroutine) == "" {
			continue
		}
		lines := strings.Split(goroutine, "\n")
		linesToPrint := []string{}
		if len(lines) > 1 {
			shouldPrint := false
			line1 := fmt.Sprintf("\nGoroutine:%s", lines[0])
			if strings.Contains(line1, "[sync.Cond.Wait]") {
				continue
			}
			linesToPrint = append(linesToPrint, line1)
			linesToPrint = append(linesToPrint, fmt.Sprintf("At line: %s", lines[1]))
			for i := 2; i < len(lines) && i < 40; i++ { // Adjust the number of lines printed here (3 lines in this example)
				if strings.Contains(lines[i], "servergolang") {
					shouldPrint = true
				}
				linesToPrint = append(linesToPrint, fmt.Sprintf("         %s", lines[i]))
			}
			if shouldPrint {
				fmt.Print(strings.Join(linesToPrint, "\n"))
			}
		}
	}
}
