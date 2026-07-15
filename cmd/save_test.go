package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSavePath(t *testing.T) {
	dir := t.TempDir()

	if err := checkSavePath(filepath.Join(dir, "out.csv")); err != nil {
		t.Errorf("valid path rejected: %v", err)
	}
	if err := checkSavePath(""); err == nil {
		t.Error("empty path accepted")
	}
	if err := checkSavePath(filepath.Join(dir, "no-such-dir", "out.csv")); err == nil {
		t.Error("missing directory accepted")
	} else if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unhelpful error: %v", err)
	}
	if err := checkSavePath(dir); err == nil {
		t.Error("directory as target accepted")
	} else if !strings.Contains(err.Error(), "directory") {
		t.Errorf("unhelpful error: %v", err)
	}
}

func TestResolveSavePathRequiresFilename(t *testing.T) {
	dir := t.TempDir()

	if _, err := resolveSavePath(dir); err == nil {
		t.Error("directory-only target accepted")
	} else if !strings.Contains(err.Error(), "include a .csv filename") {
		t.Errorf("unhelpful directory error: %v", err)
	}

	fromFile, err := resolveSavePath(filepath.Join(dir, "custom.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "custom.csv"); fromFile != want {
		t.Errorf("filename resolved to %q, want %q", fromFile, want)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	fromBare, err := resolveSavePath("historical.csv")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "historical.csv"); fromBare != want {
		t.Errorf("bare filename resolved to %q, want %q", fromBare, want)
	}

	if _, err := resolveSavePath(filepath.Join(dir, "not-csv.txt")); err == nil {
		t.Error("non-CSV filename accepted")
	}
	if _, err := resolveSavePath(filepath.Join(dir, "missing", "out.csv")); err == nil {
		t.Error("missing directory accepted")
	}
}

func TestSaveCSVOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")
	header := []string{"a", "b"}

	if err := saveCSV(path, header, [][]string{{"1", "2"}, {"3", "4"}}); err != nil {
		t.Fatal(err)
	}
	if err := saveCSV(path, header, [][]string{{"5", "6"}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := "a,b\n5,6\n"
	if string(data) != want {
		t.Errorf("got %q, want %q (second save should overwrite)", data, want)
	}
}

func TestOpenCSVAppendContinuesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ticks.csv")

	f, w, needHeader, err := openCSVAppend(path)
	if err != nil {
		t.Fatal(err)
	}
	if !needHeader {
		t.Error("new file should need a header")
	}
	w.Write([]string{"h1", "h2"})
	w.Write([]string{"1", "2"})
	w.Flush()
	f.Close()

	f, w, needHeader, err = openCSVAppend(path)
	if err != nil {
		t.Fatal(err)
	}
	if needHeader {
		t.Error("existing file should not need a header again")
	}
	w.Write([]string{"3", "4"})
	w.Flush()
	f.Close()

	data, _ := os.ReadFile(path)
	want := "h1,h2\n1,2\n3,4\n"
	if string(data) != want {
		t.Errorf("got %q, want %q", data, want)
	}
}
