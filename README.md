# filewatch

Watch a directory or file for changes.

## Usage

```go
package main

import "github.com/dmcneil/filewatch"

func main() {
	fw := filewatch.New(".") // Watch the current directory.
	defer fw.Stop()

	for {
		select {
		case <-fw.C: 
			// Do work...
		}
	}
}
```
