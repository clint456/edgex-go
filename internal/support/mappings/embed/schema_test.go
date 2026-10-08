package embed

import "testing"

func TestMigrationDirectoriesEmbedded(t *testing.T) {
	for _, directory := range []string{"sql/idempotent", "sql/versions/4.0.0-dev"} {
		entries, err := SQLFiles.ReadDir(directory)
		if err != nil || len(entries) == 0 {
			t.Fatalf("migration directory %s is unavailable: %v", directory, err)
		}
	}
}
