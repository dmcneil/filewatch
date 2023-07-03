# filewatch

Watch a directory or file for changes.

## Usage

```go
package main

import (
	"fmt"
	
	"github.com/dmcneil/filewatch"
)

func main() {
	fw := filewatch.New(".") // Watch the current directory.
	defer fw.Stop()

	for {
		select {
		case _, ok := <-fw.C:
			if !ok {
				return // Closed.
            }
			
			// Do work.
		case err, ok := <-fw.Err:
			if !ok {
				return // Closed.
			}

			fmt.Println(err)
		}
	}
}
```
