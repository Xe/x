// Command genua generates an example user agent.
package main

import (
	"fmt"

	"within.website/x/web"
)

func main() {
	fmt.Println(web.GenUserAgent())
}
