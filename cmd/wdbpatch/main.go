// Command wdbpatch reads WoW 1.12 WDB caches (the client's cache of server
// query responses) and patches the freshest values into inklab.db. The core
// lives in backend/datatools so the app's Tools tab shares it.
//
// Usage:
//
//	go run ./cmd/wdbpatch <cache.wdb> <db.sqlite>   # patch one cache
//	go run ./cmd/wdbpatch --all <wdbDir> <db.sqlite> # patch every cache in a dir
package main

import (
	"fmt"
	"os"

	"inklab/backend/datatools"
)

func main() {
	args := os.Args[1:]
	if len(args) >= 1 && args[0] == "--all" {
		if len(args) < 3 {
			fmt.Println("usage: wdbpatch --all <wdbDir> <db.sqlite>")
			os.Exit(2)
		}
		results, err := datatools.PatchAllCaches(args[1], args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, r := range results {
			report(r)
		}
		return
	}
	if len(args) < 2 {
		fmt.Println("usage: wdbpatch <cache.wdb> <db.sqlite>   (or --all <wdbDir> <db.sqlite>)")
		os.Exit(2)
	}
	report(datatools.PatchCacheFile(args[0], args[1]))
}

func report(r datatools.CacheResult) {
	if r.Error != "" {
		fmt.Printf("%-22s ERROR: %s\n", r.File, r.Error)
		return
	}
	fmt.Printf("%-22s -> %-20s records=%d updated=%d inserted=%d\n",
		r.File, r.Table, r.Records, r.Updated, r.Inserted)
}
