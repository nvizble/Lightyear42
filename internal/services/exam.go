package services

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/nvizble/Lightyear42/internal/exam"
)

// Exam session errors.
var (
	ErrNoExamSession     = errors.New("nenhuma prova em andamento; comece com `lightyear exam start` ou `lightyear exam practice <exercício>`")
	ErrExamSessionActive = errors.New("já existe uma prova em andamento; termine-a com `lightyear exam finish`")
	ErrExamTimeUp        = errors.New("tempo esgotado; encerre com `lightyear exam finish`")
)

// ExamGrader grades a submission. Implemented by exam.Grader.
type ExamGrader interface {
	Grade(ctx context.Context, ex exam.Exercise, submissionDir string) (exam.Result, error)
}

// ExamSessionStore persists the session in progress. Implemented by
// *exam.SessionFile.
type ExamSessionStore interface {
	Load() (*exam.Session, error)
	Save(exam.Session) error
	Clear() error
}

// ExamService runs exam simulations: a timed exam that climbs the levels of
// a rank, or untimed practice of a single exercise.
//
// The workspace mirrors the real exam: subjects/<ex>/ holds the statement,
// the student turns in at rendu/<ex>/, and failures leave traces/<ex>.trace.
type ExamService struct {
	catalog   []exam.Exercise
	grader    ExamGrader
	store     ExamSessionStore
	workspace string
	// pick returns a random index in [0, n); replaceable in tests.
	pick func(n int) int
}

// NewExamService wires the exercise catalog, the grader, the session store
// and the workspace directory.
func NewExamService(catalog []exam.Exercise, grader ExamGrader, store ExamSessionStore, workspace string) *ExamService {
	return &ExamService{catalog: catalog, grader: grader, store: store, workspace: workspace, pick: rand.IntN}
}

// Exercises returns the whole catalog.
func (s *ExamService) Exercises() []exam.Exercise {
	return s.catalog
}

// EditTargets returns the files an editor should open for the session's
// exercise: the subject (Portuguese when available, else English) and the
// files to turn in under rendu/.
func (s *ExamService) EditTargets(sess exam.Session) (subject string, files []string, err error) {
	ex, ok := s.find(sess.Rank, sess.Exercise)
	if !ok {
		return "", nil, fmt.Errorf("exercício %q da sessão não existe mais no catálogo", sess.Exercise)
	}
	lang := "pt"
	if _, ok := ex.Subjects[lang]; !ok {
		lang = "en"
	}
	subject = filepath.Join(sess.Workspace, "subjects", ex.Name, "subject."+lang+".txt")
	for _, name := range ex.Files {
		files = append(files, filepath.Join(sess.Workspace, "rendu", ex.Name, name))
	}
	return subject, files, nil
}

// Start begins a timed exam of the given rank at its first level.
func (s *ExamService) Start(now time.Time, rank string, duration time.Duration) (exam.Session, error) {
	levels := s.levels(rank)
	if len(levels) == 0 {
		return exam.Session{}, fmt.Errorf("rank %q não existe no catálogo", rank)
	}
	ex := s.randomExercise(rank, levels[0])
	return s.begin(now, exam.Session{
		Mode:     exam.ModeExam,
		Rank:     rank,
		Deadline: now.Add(duration),
	}, ex)
}

// Practice begins untimed practice of one exercise, by name.
func (s *ExamService) Practice(now time.Time, name string) (exam.Session, error) {
	for _, ex := range s.catalog {
		if ex.Name == name {
			return s.begin(now, exam.Session{Mode: exam.ModePractice, Rank: ex.Rank}, ex)
		}
	}
	return exam.Session{}, fmt.Errorf("exercício %q não existe; veja `lightyear exam list`", name)
}

// Status returns the session in progress.
func (s *ExamService) Status() (exam.Session, error) {
	sess, err := s.store.Load()
	if err != nil {
		return exam.Session{}, err
	}
	if sess == nil {
		return exam.Session{}, ErrNoExamSession
	}
	return *sess, nil
}

// Finish ends the session in progress and returns its final state.
// The workspace is kept, so the student's code is never deleted.
func (s *ExamService) Finish() (exam.Session, error) {
	sess, err := s.Status()
	if err != nil {
		return exam.Session{}, err
	}
	return sess, s.store.Clear()
}

// GradeReport is the outcome of one grading.
type GradeReport struct {
	exam.Result
	// Graded is the exercise that was just graded.
	Graded string
	// TracePath points to the failure trace; empty when it passed.
	TracePath string
	// Completed is true when the session ended successfully (all levels of
	// the exam, or the practiced exercise).
	Completed bool
	// Session is the state after grading (next exercise when it passed).
	Session exam.Session
}

// Grade checks the current exercise. On success the exam moves to a random
// exercise of the next level; on failure the student keeps the same one.
func (s *ExamService) Grade(ctx context.Context, now time.Time) (GradeReport, error) {
	sess, err := s.Status()
	if err != nil {
		return GradeReport{}, err
	}
	if sess.TimeUp(now) {
		return GradeReport{}, ErrExamTimeUp
	}
	ex, ok := s.find(sess.Rank, sess.Exercise)
	if !ok {
		return GradeReport{}, fmt.Errorf("exercício %q da sessão não existe mais no catálogo", sess.Exercise)
	}

	res, err := s.grader.Grade(ctx, ex, filepath.Join(sess.Workspace, "rendu", ex.Name))
	if err != nil {
		return GradeReport{}, err
	}
	sess.Attempts++
	report := GradeReport{Result: res, Graded: ex.Name}

	tracePath := filepath.Join(sess.Workspace, "traces", ex.Name+".trace")
	if !res.Passed {
		if err := writeFile(tracePath, res.Trace); err != nil {
			return GradeReport{}, err
		}
		report.TracePath = tracePath
		report.Session = sess
		return report, s.store.Save(sess)
	}
	_ = os.Remove(tracePath)

	if sess.Mode == exam.ModePractice {
		report.Completed = true
		report.Session = sess
		return report, s.store.Clear()
	}

	levels := s.levels(sess.Rank)
	passed := sort.SearchInts(levels, sess.Level) + 1
	sess.Score = 100 * passed / len(levels)
	if passed == len(levels) {
		report.Completed = true
		report.Session = sess
		return report, s.store.Clear()
	}

	next := s.randomExercise(sess.Rank, levels[passed])
	if err := s.prepare(sess.Workspace, next); err != nil {
		return GradeReport{}, err
	}
	sess.Level, sess.Exercise, sess.Attempts = next.Level, next.Name, 0
	report.Session = sess
	return report, s.store.Save(sess)
}

// begin resets the workspace, lays out the first exercise and saves the session.
func (s *ExamService) begin(now time.Time, sess exam.Session, ex exam.Exercise) (exam.Session, error) {
	active, err := s.store.Load()
	if err != nil {
		return exam.Session{}, err
	}
	if active != nil {
		return exam.Session{}, ErrExamSessionActive
	}
	if err := s.resetWorkspace(now); err != nil {
		return exam.Session{}, err
	}
	if err := s.prepare(s.workspace, ex); err != nil {
		return exam.Session{}, err
	}

	sess.Level, sess.Exercise = ex.Level, ex.Name
	sess.StartedAt, sess.Workspace = now, s.workspace
	return sess, s.store.Save(sess)
}

// resetWorkspace clears subjects/ and traces/ from a previous session and
// moves a non-empty rendu/ to history/ — the student's code is never deleted.
func (s *ExamService) resetWorkspace(now time.Time) error {
	for _, dir := range []string{"subjects", "traces"} {
		if err := os.RemoveAll(filepath.Join(s.workspace, dir)); err != nil {
			return fmt.Errorf("limpar workspace: %w", err)
		}
	}

	rendu := filepath.Join(s.workspace, "rendu")
	if entries, err := os.ReadDir(rendu); err == nil && len(entries) > 0 {
		history := filepath.Join(s.workspace, "history")
		if err := os.MkdirAll(history, 0o755); err != nil {
			return fmt.Errorf("arquivar rendu anterior: %w", err)
		}
		dest := filepath.Join(history, "rendu-"+now.Format("20060102-150405"))
		if err := os.Rename(rendu, dest); err != nil {
			return fmt.Errorf("arquivar rendu anterior: %w", err)
		}
	}
	return nil
}

// prepare writes the exercise's subjects (one file per language) and the
// files the exam provides into subjects/<ex>/, and creates its empty turn-in
// folder.
func (s *ExamService) prepare(workspace string, ex exam.Exercise) error {
	files := make(map[string]string, len(ex.Subjects)+len(ex.Provided))
	for lang, text := range ex.Subjects {
		files["subject."+lang+".txt"] = text
	}
	maps.Copy(files, ex.Provided)
	for name, content := range files {
		if err := writeFile(filepath.Join(workspace, "subjects", ex.Name, name), content); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(workspace, "rendu", ex.Name), 0o755); err != nil {
		return fmt.Errorf("criar rendu/%s: %w", ex.Name, err)
	}
	return nil
}

// levels returns the sorted, distinct levels of a rank.
func (s *ExamService) levels(rank string) []int {
	var levels []int
	seen := make(map[int]bool)
	for _, ex := range s.catalog {
		if ex.Rank == rank && !seen[ex.Level] {
			seen[ex.Level] = true
			levels = append(levels, ex.Level)
		}
	}
	sort.Ints(levels)
	return levels
}

func (s *ExamService) randomExercise(rank string, level int) exam.Exercise {
	var pool []exam.Exercise
	for _, ex := range s.catalog {
		if ex.Rank == rank && ex.Level == level {
			pool = append(pool, ex)
		}
	}
	return pool[s.pick(len(pool))]
}

func (s *ExamService) find(rank, name string) (exam.Exercise, bool) {
	for _, ex := range s.catalog {
		if ex.Rank == rank && ex.Name == name {
			return ex, true
		}
	}
	return exam.Exercise{}, false
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("criar %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("gravar %s: %w", path, err)
	}
	return nil
}
