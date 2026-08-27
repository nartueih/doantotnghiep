package migrations

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Migration struct {
	Version int
	Name    string
	SQL     string
}

const LatestVersion = 4

//go:embed *.sql
var files embed.FS

func All() []Migration {
	entries, err := files.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("read embedded migrations: %v", err))
	}

	items := make([]Migration, 0, len(entries))
	seen := make(map[int]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			panic(err)
		}
		if existing, exists := seen[version]; exists {
			panic(fmt.Sprintf("duplicate migration version %03d in %s and %s", version, existing, entry.Name()))
		}
		contents, err := files.ReadFile(entry.Name())
		if err != nil {
			panic(fmt.Sprintf("read embedded migration %s: %v", entry.Name(), err))
		}
		seen[version] = entry.Name()
		items = append(items, Migration{Version: version, Name: entry.Name(), SQL: string(contents)})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Version < items[j].Version })
	return items
}

func parseVersion(name string) (int, error) {
	separator := strings.IndexByte(name, '_')
	if separator != 3 || !strings.HasSuffix(name, ".sql") {
		return 0, fmt.Errorf("invalid migration filename %q; expected NNN_name.sql", name)
	}
	version, err := strconv.Atoi(name[:separator])
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("invalid migration version in %q", name)
	}
	return version, nil
}
