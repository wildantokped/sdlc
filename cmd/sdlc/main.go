package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wildantokped/sdlc"
)

func run() error {
	repoPath := os.Args[1]
	gitApp, err := sdlc.NewGitAppFromDir(repoPath)
	if err != nil {
		return errors.Wrap(err, "new git app from dir")
	}

	schemaScript, err := gitApp.GenerateSchemaScript()
	if err != nil {
		return errors.Wrap(err, "generate schema script")
	}

	changelogs, err := gitApp.GetLatestChangeLogs()
	if err != nil {
		return errors.Wrap(err, "get latest changelogs")
	}

	var markdownResult strings.Builder
	serviceName := filepath.Base(repoPath)
	markdownResult.WriteString("**Service**: \n* " + serviceName + "\n\n")
	markdownResult.WriteString(changelogs.String())

	markdownResult.WriteString("## SQL Scripts\n")
	markdownResult.WriteString("```sql\n")
	markdownResult.WriteString(schemaScript)
	markdownResult.WriteString("\n```")

	target, err := os.Create(fmt.Sprintf("CHANGELOG-%d.md", time.Now().Unix()))
	if err != nil {
		return errors.Wrap(err, "create target file")
	}
	defer target.Close()

	target.WriteString(markdownResult.String())

	return nil
}

func main() {
	start := time.Now()
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("generated in %s", time.Since(start))
}
