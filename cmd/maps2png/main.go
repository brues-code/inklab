// Command maps2png stitches WoW client world-map BLP tiles into per-zone JPGs.
// The decoder/stitcher lives in backend/datatools so the app's Tools tab can
// run the same job in-process.
//
// Usage:
//
//	go run ./cmd/maps2png <Interface/WorldMap dir> <out dir>
package main

import (
	"fmt"
	"os"

	"inklab/backend/datatools"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: maps2png <WorldMap dir> <out dir>")
		os.Exit(2)
	}
	res, err := datatools.GenerateZoneMaps(os.Args[1], os.Args[2], func(zone string, i, total int) {
		fmt.Printf("\r[%d/%d] %-30s", i, total, zone)
	})
	fmt.Println()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("generated %d, skipped %d\n", res.Generated, res.Skipped)
	for _, w := range res.Warnings {
		fmt.Println("  warn:", w)
	}
}
