package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nvizble/Lightyear42/internal/exam"
)

type fakeGrader struct{ pass bool }

func (g *fakeGrader) Grade(context.Context, exam.Exercise, string) (exam.Result, error) {
	if g.pass {
		return exam.Result{Passed: true}, nil
	}
	return exam.Result{Trace: "esperado: \"a\""}, nil
}

type memSessionStore struct{ sess *exam.Session }

func (m *memSessionStore) Load() (*exam.Session, error) { return m.sess, nil }
func (m *memSessionStore) Save(s exam.Session) error    { m.sess = &s; return nil }
func (m *memSessionStore) Clear() error                 { m.sess = nil; return nil }

var examNow = time.Date(2026, 10, 6, 14, 0, 0, 0, time.UTC)

func newTestExamService(t *testing.T) (*ExamService, *fakeGrader, *memSessionStore) {
	t.Helper()
	catalog := []exam.Exercise{
		{Rank: "02", Level: 2, Name: "union", Subjects: map[string]string{"en": "union", "pt": "união"}, Provided: map[string]string{"list.h": "struct"}},
		{Rank: "02", Level: 1, Name: "first_word", Subjects: map[string]string{"en": "fw"}},
		{Rank: "03", Level: 1, Name: "other", Subjects: map[string]string{"en": "x"}},
	}
	grader, store := &fakeGrader{}, &memSessionStore{}
	svc := NewExamService(catalog, grader, store, t.TempDir())
	svc.pick = func(int) int { return 0 }
	return svc, grader, store
}

func TestExamStartLaysOutFirstLevel(t *testing.T) {
	svc, _, store := newTestExamService(t)

	sess, err := svc.Start(examNow, "02", 3*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Exercise != "first_word" || sess.Level != 1 || sess.Score != 0 {
		t.Fatalf("sessão inesperada: %+v", sess)
	}
	if store.sess == nil || !store.sess.Deadline.Equal(examNow.Add(3*time.Hour)) {
		t.Fatalf("sessão não salva com deadline: %+v", store.sess)
	}
	assertFile(t, filepath.Join(sess.Workspace, "subjects", "first_word", "subject.en.txt"), "fw")
	if _, err := os.Stat(filepath.Join(sess.Workspace, "rendu", "first_word")); err != nil {
		t.Fatalf("rendu não criado: %v", err)
	}

	if _, err := svc.Start(examNow, "02", time.Hour); !errors.Is(err, ErrExamSessionActive) {
		t.Fatalf("esperava ErrExamSessionActive, veio %v", err)
	}
	if _, err := svc.Start(examNow, "99", time.Hour); err == nil {
		t.Fatal("rank inexistente deveria falhar")
	}
}

func TestExamGradeProgression(t *testing.T) {
	svc, grader, store := newTestExamService(t)
	sess, err := svc.Start(examNow, "02", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Failure: same exercise, attempt counted, trace written.
	report, err := svc.Grade(context.Background(), examNow)
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || report.Session.Exercise != "first_word" || report.Session.Attempts != 1 {
		t.Fatalf("falha mal registrada: %+v", report)
	}
	assertFile(t, report.TracePath, "esperado: \"a\"")

	// Pass level 1 of 2: half the score, next level laid out.
	grader.pass = true
	report, err = svc.Grade(context.Background(), examNow)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Completed || report.Session.Exercise != "union" || report.Session.Score != 50 || report.Session.Attempts != 0 {
		t.Fatalf("progressão errada: %+v", report)
	}
	assertFile(t, filepath.Join(sess.Workspace, "subjects", "union", "subject.pt.txt"), "união")
	assertFile(t, filepath.Join(sess.Workspace, "subjects", "union", "list.h"), "struct")

	// Pass the last level: exam completed and session cleared.
	report, err = svc.Grade(context.Background(), examNow)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Completed || report.Session.Score != 100 || store.sess != nil {
		t.Fatalf("conclusão errada: %+v, store=%+v", report, store.sess)
	}
	if _, err := svc.Grade(context.Background(), examNow); !errors.Is(err, ErrNoExamSession) {
		t.Fatalf("esperava ErrNoExamSession, veio %v", err)
	}
}

func TestExamGradeAfterDeadline(t *testing.T) {
	svc, _, _ := newTestExamService(t)
	if _, err := svc.Start(examNow, "02", time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Grade(context.Background(), examNow.Add(time.Hour)); !errors.Is(err, ErrExamTimeUp) {
		t.Fatalf("esperava ErrExamTimeUp, veio %v", err)
	}
	sess, err := svc.Finish()
	if err != nil || sess.Exercise != "first_word" {
		t.Fatalf("finish: %+v, %v", sess, err)
	}
}

func TestExamPractice(t *testing.T) {
	svc, grader, store := newTestExamService(t)
	sess, err := svc.Practice(examNow, "union")
	if err != nil {
		t.Fatal(err)
	}
	if sess.Mode != exam.ModePractice || !sess.Deadline.IsZero() || sess.Level != 2 {
		t.Fatalf("prática inesperada: %+v", sess)
	}
	if sess.TimeUp(examNow.Add(100 * time.Hour)) {
		t.Fatal("prática não tem tempo limite")
	}

	grader.pass = true
	report, err := svc.Grade(context.Background(), examNow)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Completed || store.sess != nil {
		t.Fatalf("prática deveria terminar ao passar: %+v", report)
	}
	if _, err := svc.Practice(examNow, "nope"); err == nil {
		t.Fatal("exercício inexistente deveria falhar")
	}
}

func TestExamStartArchivesPreviousRendu(t *testing.T) {
	svc, _, _ := newTestExamService(t)
	old := filepath.Join(svc.workspace, "rendu", "first_word", "first_word.c")
	if err := writeFile(old, "int main(void){}"); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Start(examNow, "02", time.Hour); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(svc.workspace, "history", "rendu-20261006-140000", "first_word", "first_word.c")
	assertFile(t, archived, "int main(void){}")
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("rendu antigo deveria ter saído do lugar: %v", err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, esperado %q", path, got, want)
	}
}
