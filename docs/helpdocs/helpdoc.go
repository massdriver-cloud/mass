package helpdocs

import (
	"embed"
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	//go:embed *
	helpdocs embed.FS

	ignoreGlamour bool
	once          sync.Once
)

func MustRender(name string) string {
	path := fmt.Sprintf("%s.md", name)
	data, err := helpdocs.ReadFile(path)
	if err != nil {
		panic(err)
	}

	// Once is enough for this, it won't change after the first call
	once.Do(func() {
		if len(os.Args) > 1 && os.Args[1] == "docs" {
			ignoreGlamour = true
		}
	})

	// In order to use cobra doc gen, we need the raw .md file
	// without the glamour rendering
	if ignoreGlamour {
		return string(data)
	}

	out, err := glamour.Render(string(data), "auto")
	if err != nil {
		panic(err)
	}
	return out
}
