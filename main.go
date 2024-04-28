/*
Copyright © 2024 Ryan Painter paintersrp@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type TaskFile struct {
	Path      string
	TaskCount int
}

func parseMarkdownFile(
	path string,
	taskMap *map[string][]string,
) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	parser := goldmark.DefaultParser()
	reader := text.NewReader(source)
	document := parser.Parse(reader)

	var tasks []string

	ast.Walk(
		document,
		func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering {
				if listItem, ok := n.(*ast.ListItem); ok {
					content := strings.TrimSpace(
						string(listItem.Text(source)),
					)
					if (strings.HasPrefix(content, "[ ]") || strings.HasPrefix(content, "[x]")) &&
						len(strings.TrimSpace(content[3:])) > 0 {
						tasks = append(tasks, content)
					}
				}
			}
			return ast.WalkContinue, nil
		},
	)

	(*taskMap)[path] = tasks

	return nil
}

func walkDir(dirPath string, taskMap *map[string][]string) error {
	return filepath.Walk(
		dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf(
					"error walking the path %q: %w",
					path,
					err,
				)
			}
			if !info.IsDir() && filepath.Ext(path) == ".md" {
				if err := parseMarkdownFile(path, taskMap); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func main() {
	home := "/home/srp/note-test"
	taskMap := make(map[string][]string)

	if err := walkDir(home, &taskMap); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Convert the map to a slice of TaskFile for sorting.
	var taskFiles []TaskFile
	for file, tasks := range taskMap {
		if len(tasks) > 0 { // Only include files with tasks.
			taskFiles = append(
				taskFiles,
				TaskFile{Path: file, TaskCount: len(tasks)},
			)
		}
	}

	// Sort the slice by TaskCount in descending order.
	sort.Slice(taskFiles, func(i, j int) bool {
		return taskFiles[i].TaskCount > taskFiles[j].TaskCount
	})

	// Print the sorted list of files and their task count.
	for _, tf := range taskFiles {
		fmt.Printf(
			"File: %s, Task Count: %d\n",
			tf.Path,
			tf.TaskCount,
		)
	}

	// Optionally, print all tasks.
	fmt.Println("\nAll Tasks:")
	for _, file := range taskFiles {
		for _, task := range taskMap[file.Path] {
			fmt.Println(task)
		}
	}
}

// func main() {
// 	cmd.Execute()
// }

// func main() {
// 	templater.Init()
// }
