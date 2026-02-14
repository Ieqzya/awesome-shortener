package main

import "os"

func main() {
	os.Exit(1) // want "прямой вызов os.Exit в функции main запрещен"
}

func helper() {
	os.Exit(0) // это разрешено, так как не в main
}
