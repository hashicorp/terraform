package api

import (
	"testing"
)

func assertQueryMeta(t *testing.T, qm *QueryMeta) {
	if qm.LastIndex == 0 {
		t.Fatalf("bad index: %d", qm.LastIndex)
	}
	if !qm.KnownLeader {
		t.Fatalf("expected known leader, got none")
	}
}

func assertWriteMeta(t *testing.T, wm *WriteMeta) {
	if wm.LastIndex == 0 {
		t.Fatalf("bad index: %d", wm.LastIndex)
	}
}

func testJob() *Job {
	task := NewTask("task1", "exec").
		SetConfig("command", "/bin/sleep").
		Require(&Resources{
			CPU:      100,
			MemoryMB: 256,
			DiskMB:   25,
			IOPS:     10,
		}).
		SetLogConfig(&LogConfig{
			MaxFiles:      1,
			MaxFileSizeMB: 2,
		})

	group := NewTaskGroup("group1", 1).
		AddTask(task)

	job := NewBatchJob("job1", "redis", "region1", 1).
		AddDatacenter("dc1").
		AddTaskGroup(group)

	return job
}

func testPeriodicJob() *Job {
	job := testJob().AddPeriodicConfig(&PeriodicConfig{
		Enabled:  true,
		Spec:     "*/30 * * * *",
		SpecType: "cron",
	})
	return job
}
