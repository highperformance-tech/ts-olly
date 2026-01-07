package fileid

import (
	"errors"
	"os"
	"testing"
)

func TestFileID(t *testing.T) {
	t.Run("renaming file keeps id", func(t *testing.T) {
		dir := os.TempDir()

		file1 := dir + "/fileid_test"
		f, err := os.Create(file1)
		if err != nil {
			t.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			t.Fatal(err)
		}

		id, err := Query(file1)
		if err != nil {
			t.Fatal(err)
		}

		file2 := dir + "/fileid_test2"
		if err = os.Rename(file1, file2); err != nil {
			t.Fatal(err)
		}

		id2, err := Query(file2)
		if err != nil {
			t.Fatal(err)
		}

		if id != id2 {
			t.Errorf("file ID changed")
		}
	})

	t.Run("missing file returns NotExist error", func(t *testing.T) {
		_, err := Query("/tmp/fileid_test")
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected error")
		}
	})
}
