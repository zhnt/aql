package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/zhnt/aql/internal/gc"
)

func main() {
	fmt.Println("AQL Memory Allocator Demo")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU count: %d\n", runtime.NumCPU())
	fmt.Println()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "usage":
			gc.ExampleAllocatorUsage()
		case "config":
			gc.ExampleConfigurationTuning()
		case "monitor":
			gc.ExampleMemoryUsageMonitoring()
		case "all":
			gc.ExampleAllocatorUsage()
			fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
			gc.ExampleConfigurationTuning()
			fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
			gc.ExampleMemoryUsageMonitoring()
		default:
			printUsage()
		}
	} else {
		// 默认运行基本演示
		gc.ExampleAllocatorUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: allocator-demo [demo-type]")
	fmt.Println()
	fmt.Println("Demo types:")
	fmt.Println("  usage     - Basic allocator usage demonstration")
	fmt.Println("  config    - Configuration tuning examples")
	fmt.Println("  monitor   - Memory usage monitoring examples")
	fmt.Println("  all       - Run all demonstrations")
	fmt.Println()
	fmt.Println("If no demo type is specified, 'usage' is run by default.")
}
