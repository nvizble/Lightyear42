// Package exam holds the embedded exam exercises and the grader that checks
// a submission against each exercise's reference solution.
package exam

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed exercises
var exercisesFS embed.FS

// Exercise is one exam assignment.
//
// On disk it lives at exercises/rank<NN>/level<N>/<name>/ with:
//   - subject.<lang>.txt: the statement, one file per language (en required)
//   - meta.json:          files to turn in and the argv of each test
//   - ref/:               the reference solution, laid out exactly like a
//     correct submission (same file names as meta.json "files")
//   - provided/:          optional files the exam hands out (e.g. a header); they
//     go into both builds and next to the subject, and the student doesn't turn them in
//   - main.c:             optional harness for function exercises (no main of their own)
type Exercise struct {
	Rank  string
	Level int
	Name  string
	// Subjects maps a language code ("en", "pt") to the statement.
	Subjects map[string]string
	// Files the student must turn in, relative to the exercise folder.
	Files []string
	// Tests lists the argv (without program name) of each test run.
	Tests [][]string
	// Provided maps file name to content for files handed out by the exam.
	Provided map[string]string

	ref     map[string]string
	harness string
}

type exerciseMeta struct {
	Files []string   `json:"files"`
	Tests [][]string `json:"tests"`
}

// Load returns every embedded exercise, ordered by rank, level and name.
func Load() ([]Exercise, error) {
	return load(exercisesFS, "exercises")
}

func load(fsys fs.FS, root string) ([]Exercise, error) {
	dirs, err := fs.Glob(fsys, path.Join(root, "rank*", "level*", "*"))
	if err != nil {
		return nil, err
	}

	exercises := make([]Exercise, 0, len(dirs))
	for _, dir := range dirs {
		ex, err := loadExercise(fsys, dir)
		if err != nil {
			return nil, fmt.Errorf("exercício %s: %w", dir, err)
		}
		exercises = append(exercises, ex)
	}

	sort.Slice(exercises, func(i, j int) bool {
		a, b := exercises[i], exercises[j]
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		return a.Name < b.Name
	})
	return exercises, nil
}

func loadExercise(fsys fs.FS, dir string) (Exercise, error) {
	levelDir := path.Dir(dir)
	ex := Exercise{Name: path.Base(dir)}
	if _, err := fmt.Sscanf(path.Base(path.Dir(levelDir)), "rank%s", &ex.Rank); err != nil {
		return Exercise{}, fmt.Errorf("rank inválido: %w", err)
	}
	if _, err := fmt.Sscanf(path.Base(levelDir), "level%d", &ex.Level); err != nil {
		return Exercise{}, fmt.Errorf("level inválido: %w", err)
	}

	read := func(name string) (string, error) {
		data, err := fs.ReadFile(fsys, path.Join(dir, name))
		return string(data), err
	}

	subjects, err := fs.Glob(fsys, path.Join(dir, "subject.*.txt"))
	if err != nil {
		return Exercise{}, err
	}
	ex.Subjects = make(map[string]string, len(subjects))
	for _, name := range subjects {
		lang := strings.TrimSuffix(strings.TrimPrefix(path.Base(name), "subject."), ".txt")
		if ex.Subjects[lang], err = read(path.Base(name)); err != nil {
			return Exercise{}, err
		}
	}
	if ex.Subjects["en"] == "" {
		return Exercise{}, fmt.Errorf("falta subject.en.txt")
	}

	if ex.ref, err = readDir(fsys, path.Join(dir, "ref")); err != nil {
		return Exercise{}, err
	}
	if ex.Provided, err = readDir(fsys, path.Join(dir, "provided")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Exercise{}, err
	}
	if ex.harness, err = read("main.c"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Exercise{}, err
	}

	raw, err := read("meta.json")
	if err != nil {
		return Exercise{}, err
	}
	var meta exerciseMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return Exercise{}, fmt.Errorf("meta.json: %w", err)
	}
	if len(meta.Files) == 0 || len(meta.Tests) == 0 {
		return Exercise{}, fmt.Errorf("meta.json precisa de files e tests")
	}
	ex.Files, ex.Tests = meta.Files, meta.Tests
	return ex, nil
}

// readDir returns name → content for every file in dir.
func readDir(fsys fs.FS, dir string) (map[string]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string, len(entries))
	for _, e := range entries {
		data, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = string(data)
	}
	return files, nil
}
