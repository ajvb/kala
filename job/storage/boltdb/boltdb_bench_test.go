package boltdb

import (
	"testing"
	"time"

	"github.com/ajvb/kala/v2/job"
)

// BenchmarkSaveJobWithLargeHistory measures saving a job that has 1,000 runs
// in its history, which is stored with the job. The database does not sync to
// disk after each save, so the result measures encoding and writing the job
// rather than the speed of the machine's disk.
func BenchmarkSaveJobWithLargeHistory(b *testing.B) {
	db := GetBoltDB(b.TempDir())
	defer db.Close()
	db.dbConn.NoSync = true

	j := job.GetMockJobWithGenericSchedule(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	j.Id = "benchmark-job"
	ranAt := time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 1000 {
		stat := job.NewJobStat(j.Id, ranAt.Add(time.Duration(i)*time.Minute))
		stat.Success = true
		stat.ExecutionDuration = time.Second
		j.Stats = append(j.Stats, stat)
	}

	for b.Loop() {
		if err := db.Save(j); err != nil {
			b.Fatal(err)
		}
	}
}
