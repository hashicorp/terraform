package structs

import (
	"reflect"
	"testing"
	"time"
)

func TestJobDiff(t *testing.T) {
	cases := []struct {
		Old, New   *Job
		Expected   *JobDiff
		Error      bool
		Contextual bool
	}{
		{
			Old: nil,
			New: nil,
			Expected: &JobDiff{
				Type: DiffTypeNone,
			},
		},
		{
			// Different IDs
			Old: &Job{
				ID: "foo",
			},
			New: &Job{
				ID: "bar",
			},
			Error: true,
		},
		{
			// Primitive only that is the same
			Old: &Job{
				Region:    "foo",
				ID:        "foo",
				Name:      "foo",
				Type:      "batch",
				Priority:  10,
				AllAtOnce: true,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: &Job{
				Region:    "foo",
				ID:        "foo",
				Name:      "foo",
				Type:      "batch",
				Priority:  10,
				AllAtOnce: true,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeNone,
				ID:   "foo",
			},
		},
		{
			// Primitive only that is has diffs
			Old: &Job{
				Region:    "foo",
				ID:        "foo",
				Name:      "foo",
				Type:      "batch",
				Priority:  10,
				AllAtOnce: true,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: &Job{
				Region:    "bar",
				ID:        "foo",
				Name:      "bar",
				Type:      "system",
				Priority:  100,
				AllAtOnce: false,
				Meta: map[string]string{
					"foo": "baz",
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				ID:   "foo",
				Fields: []*FieldDiff{
					{
						Type: DiffTypeEdited,
						Name: "AllAtOnce",
						Old:  "true",
						New:  "false",
					},
					{
						Type: DiffTypeEdited,
						Name: "Meta[foo]",
						Old:  "bar",
						New:  "baz",
					},
					{
						Type: DiffTypeEdited,
						Name: "Name",
						Old:  "foo",
						New:  "bar",
					},
					{
						Type: DiffTypeEdited,
						Name: "Priority",
						Old:  "10",
						New:  "100",
					},
					{
						Type: DiffTypeEdited,
						Name: "Region",
						Old:  "foo",
						New:  "bar",
					},
					{
						Type: DiffTypeEdited,
						Name: "Type",
						Old:  "batch",
						New:  "system",
					},
				},
			},
		},
		{
			// Primitive only deleted job
			Old: &Job{
				Region:    "foo",
				ID:        "foo",
				Name:      "foo",
				Type:      "batch",
				Priority:  10,
				AllAtOnce: true,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: nil,
			Expected: &JobDiff{
				Type: DiffTypeDeleted,
				ID:   "foo",
				Fields: []*FieldDiff{
					{
						Type: DiffTypeDeleted,
						Name: "AllAtOnce",
						Old:  "true",
						New:  "",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Meta[foo]",
						Old:  "bar",
						New:  "",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Name",
						Old:  "foo",
						New:  "",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Priority",
						Old:  "10",
						New:  "",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Region",
						Old:  "foo",
						New:  "",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Type",
						Old:  "batch",
						New:  "",
					},
				},
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeDeleted,
						Name: "Update",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "MaxParallel",
								Old:  "0",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Stagger",
								Old:  "0",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Primitive only added job
			Old: nil,
			New: &Job{
				Region:    "foo",
				ID:        "foo",
				Name:      "foo",
				Type:      "batch",
				Priority:  10,
				AllAtOnce: true,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeAdded,
				ID:   "foo",
				Fields: []*FieldDiff{
					{
						Type: DiffTypeAdded,
						Name: "AllAtOnce",
						Old:  "",
						New:  "true",
					},
					{
						Type: DiffTypeAdded,
						Name: "Meta[foo]",
						Old:  "",
						New:  "bar",
					},
					{
						Type: DiffTypeAdded,
						Name: "Name",
						Old:  "",
						New:  "foo",
					},
					{
						Type: DiffTypeAdded,
						Name: "Priority",
						Old:  "",
						New:  "10",
					},
					{
						Type: DiffTypeAdded,
						Name: "Region",
						Old:  "",
						New:  "foo",
					},
					{
						Type: DiffTypeAdded,
						Name: "Type",
						Old:  "",
						New:  "batch",
					},
				},
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Update",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "MaxParallel",
								Old:  "",
								New:  "0",
							},
							{
								Type: DiffTypeAdded,
								Name: "Stagger",
								Old:  "",
								New:  "0",
							},
						},
					},
				},
			},
		},
		{
			// Map diff
			Old: &Job{
				Meta: map[string]string{
					"foo": "foo",
					"bar": "bar",
				},
			},
			New: &Job{
				Meta: map[string]string{
					"bar": "bar",
					"baz": "baz",
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Fields: []*FieldDiff{
					{
						Type: DiffTypeAdded,
						Name: "Meta[baz]",
						Old:  "",
						New:  "baz",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Meta[foo]",
						Old:  "foo",
						New:  "",
					},
				},
			},
		},
		{
			// Datacenter diff both added and removed
			Old: &Job{
				Datacenters: []string{"foo", "bar"},
			},
			New: &Job{
				Datacenters: []string{"baz", "bar"},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Datacenters",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Datacenters",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Datacenters",
								Old:  "foo",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Datacenter diff just added
			Old: &Job{
				Datacenters: []string{"foo", "bar"},
			},
			New: &Job{
				Datacenters: []string{"foo", "bar", "baz"},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Datacenters",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Datacenters",
								Old:  "",
								New:  "baz",
							},
						},
					},
				},
			},
		},
		{
			// Datacenter diff just deleted
			Old: &Job{
				Datacenters: []string{"foo", "bar"},
			},
			New: &Job{
				Datacenters: []string{"foo"},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeDeleted,
						Name: "Datacenters",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "Datacenters",
								Old:  "bar",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Update strategy edited
			Old: &Job{
				Update: UpdateStrategy{
					Stagger:     10 * time.Second,
					MaxParallel: 5,
				},
			},
			New: &Job{
				Update: UpdateStrategy{
					Stagger:     60 * time.Second,
					MaxParallel: 10,
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Update",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "MaxParallel",
								Old:  "5",
								New:  "10",
							},
							{
								Type: DiffTypeEdited,
								Name: "Stagger",
								Old:  "10000000000",
								New:  "60000000000",
							},
						},
					},
				},
			},
		},
		{
			// Update strategy edited with context
			Contextual: true,
			Old: &Job{
				Update: UpdateStrategy{
					Stagger:     10 * time.Second,
					MaxParallel: 5,
				},
			},
			New: &Job{
				Update: UpdateStrategy{
					Stagger:     60 * time.Second,
					MaxParallel: 5,
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Update",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeNone,
								Name: "MaxParallel",
								Old:  "5",
								New:  "5",
							},
							{
								Type: DiffTypeEdited,
								Name: "Stagger",
								Old:  "10000000000",
								New:  "60000000000",
							},
						},
					},
				},
			},
		},
		{
			// Periodic added
			Old: &Job{},
			New: &Job{
				Periodic: &PeriodicConfig{
					Enabled:         false,
					Spec:            "*/15 * * * * *",
					SpecType:        "foo",
					ProhibitOverlap: false,
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Periodic",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Enabled",
								Old:  "",
								New:  "false",
							},
							{
								Type: DiffTypeAdded,
								Name: "ProhibitOverlap",
								Old:  "",
								New:  "false",
							},
							{
								Type: DiffTypeAdded,
								Name: "Spec",
								Old:  "",
								New:  "*/15 * * * * *",
							},
							{
								Type: DiffTypeAdded,
								Name: "SpecType",
								Old:  "",
								New:  "foo",
							},
						},
					},
				},
			},
		},
		{
			// Periodic deleted
			Old: &Job{
				Periodic: &PeriodicConfig{
					Enabled:         false,
					Spec:            "*/15 * * * * *",
					SpecType:        "foo",
					ProhibitOverlap: false,
				},
			},
			New: &Job{},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeDeleted,
						Name: "Periodic",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "Enabled",
								Old:  "false",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "ProhibitOverlap",
								Old:  "false",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Spec",
								Old:  "*/15 * * * * *",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "SpecType",
								Old:  "foo",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Periodic edited
			Old: &Job{
				Periodic: &PeriodicConfig{
					Enabled:         false,
					Spec:            "*/15 * * * * *",
					SpecType:        "foo",
					ProhibitOverlap: false,
				},
			},
			New: &Job{
				Periodic: &PeriodicConfig{
					Enabled:         true,
					Spec:            "* * * * * *",
					SpecType:        "cron",
					ProhibitOverlap: true,
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Periodic",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "Enabled",
								Old:  "false",
								New:  "true",
							},
							{
								Type: DiffTypeEdited,
								Name: "ProhibitOverlap",
								Old:  "false",
								New:  "true",
							},
							{
								Type: DiffTypeEdited,
								Name: "Spec",
								Old:  "*/15 * * * * *",
								New:  "* * * * * *",
							},
							{
								Type: DiffTypeEdited,
								Name: "SpecType",
								Old:  "foo",
								New:  "cron",
							},
						},
					},
				},
			},
		},
		{
			// Periodic edited with context
			Contextual: true,
			Old: &Job{
				Periodic: &PeriodicConfig{
					Enabled:         false,
					Spec:            "*/15 * * * * *",
					SpecType:        "foo",
					ProhibitOverlap: false,
				},
			},
			New: &Job{
				Periodic: &PeriodicConfig{
					Enabled:         true,
					Spec:            "* * * * * *",
					SpecType:        "foo",
					ProhibitOverlap: false,
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Periodic",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "Enabled",
								Old:  "false",
								New:  "true",
							},
							{
								Type: DiffTypeNone,
								Name: "ProhibitOverlap",
								Old:  "false",
								New:  "false",
							},
							{
								Type: DiffTypeEdited,
								Name: "Spec",
								Old:  "*/15 * * * * *",
								New:  "* * * * * *",
							},
							{
								Type: DiffTypeNone,
								Name: "SpecType",
								Old:  "foo",
								New:  "foo",
							},
						},
					},
				},
			},
		},
		{
			// Constraints edited
			Old: &Job{
				Constraints: []*Constraint{
					{
						LTarget: "foo",
						RTarget: "foo",
						Operand: "foo",
						str:     "foo",
					},
					{
						LTarget: "bar",
						RTarget: "bar",
						Operand: "bar",
						str:     "bar",
					},
				},
			},
			New: &Job{
				Constraints: []*Constraint{
					{
						LTarget: "foo",
						RTarget: "foo",
						Operand: "foo",
						str:     "foo",
					},
					{
						LTarget: "baz",
						RTarget: "baz",
						Operand: "baz",
						str:     "baz",
					},
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Constraint",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "LTarget",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "Operand",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "RTarget",
								Old:  "",
								New:  "baz",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "Constraint",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "LTarget",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Operand",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "RTarget",
								Old:  "bar",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Task groups edited
			Old: &Job{
				TaskGroups: []*TaskGroup{
					{
						Name:  "foo",
						Count: 1,
					},
					{
						Name:  "bar",
						Count: 1,
					},
					{
						Name:  "baz",
						Count: 1,
					},
				},
			},
			New: &Job{
				TaskGroups: []*TaskGroup{
					{
						Name:  "bar",
						Count: 1,
					},
					{
						Name:  "baz",
						Count: 2,
					},
					{
						Name:  "bam",
						Count: 1,
					},
				},
			},
			Expected: &JobDiff{
				Type: DiffTypeEdited,
				TaskGroups: []*TaskGroupDiff{
					{
						Type: DiffTypeAdded,
						Name: "bam",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Count",
								Old:  "",
								New:  "1",
							},
						},
					},
					{
						Type: DiffTypeNone,
						Name: "bar",
					},
					{
						Type: DiffTypeEdited,
						Name: "baz",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "Count",
								Old:  "1",
								New:  "2",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "foo",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "Count",
								Old:  "1",
								New:  "",
							},
						},
					},
				},
			},
		},
	}

	for i, c := range cases {
		actual, err := c.Old.Diff(c.New, c.Contextual)
		if c.Error && err == nil {
			t.Fatalf("case %d: expected errored", i+1)
		} else if err != nil {
			if !c.Error {
				t.Fatalf("case %d: errored %#v", i+1, err)
			} else {
				continue
			}
		}

		if !reflect.DeepEqual(actual, c.Expected) {
			t.Fatalf("case %d: got:\n%#v\n want:\n%#v\n",
				i+1, actual, c.Expected)
		}
	}
}

func TestTaskGroupDiff(t *testing.T) {
	cases := []struct {
		Old, New   *TaskGroup
		Expected   *TaskGroupDiff
		Error      bool
		Contextual bool
	}{
		{
			Old: nil,
			New: nil,
			Expected: &TaskGroupDiff{
				Type: DiffTypeNone,
			},
		},
		{
			// Primitive only that has different names
			Old: &TaskGroup{
				Name:  "foo",
				Count: 10,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: &TaskGroup{
				Name:  "bar",
				Count: 10,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			Error: true,
		},
		{
			// Primitive only that is the same
			Old: &TaskGroup{
				Name:  "foo",
				Count: 10,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: &TaskGroup{
				Name:  "foo",
				Count: 10,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeNone,
				Name: "foo",
			},
		},
		{
			// Primitive only that has diffs
			Old: &TaskGroup{
				Name:  "foo",
				Count: 10,
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: &TaskGroup{
				Name:  "foo",
				Count: 100,
				Meta: map[string]string{
					"foo": "baz",
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Name: "foo",
				Fields: []*FieldDiff{
					{
						Type: DiffTypeEdited,
						Name: "Count",
						Old:  "10",
						New:  "100",
					},
					{
						Type: DiffTypeEdited,
						Name: "Meta[foo]",
						Old:  "bar",
						New:  "baz",
					},
				},
			},
		},
		{
			// Map diff
			Old: &TaskGroup{
				Meta: map[string]string{
					"foo": "foo",
					"bar": "bar",
				},
			},
			New: &TaskGroup{
				Meta: map[string]string{
					"bar": "bar",
					"baz": "baz",
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Fields: []*FieldDiff{
					{
						Type: DiffTypeAdded,
						Name: "Meta[baz]",
						Old:  "",
						New:  "baz",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Meta[foo]",
						Old:  "foo",
						New:  "",
					},
				},
			},
		},
		{
			// Constraints edited
			Old: &TaskGroup{
				Constraints: []*Constraint{
					{
						LTarget: "foo",
						RTarget: "foo",
						Operand: "foo",
						str:     "foo",
					},
					{
						LTarget: "bar",
						RTarget: "bar",
						Operand: "bar",
						str:     "bar",
					},
				},
			},
			New: &TaskGroup{
				Constraints: []*Constraint{
					{
						LTarget: "foo",
						RTarget: "foo",
						Operand: "foo",
						str:     "foo",
					},
					{
						LTarget: "baz",
						RTarget: "baz",
						Operand: "baz",
						str:     "baz",
					},
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Constraint",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "LTarget",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "Operand",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "RTarget",
								Old:  "",
								New:  "baz",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "Constraint",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "LTarget",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Operand",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "RTarget",
								Old:  "bar",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// RestartPolicy added
			Old: &TaskGroup{},
			New: &TaskGroup{
				RestartPolicy: &RestartPolicy{
					Attempts: 1,
					Interval: 1 * time.Second,
					Delay:    1 * time.Second,
					Mode:     "fail",
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "RestartPolicy",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Attempts",
								Old:  "",
								New:  "1",
							},
							{
								Type: DiffTypeAdded,
								Name: "Delay",
								Old:  "",
								New:  "1000000000",
							},
							{
								Type: DiffTypeAdded,
								Name: "Interval",
								Old:  "",
								New:  "1000000000",
							},
							{
								Type: DiffTypeAdded,
								Name: "Mode",
								Old:  "",
								New:  "fail",
							},
						},
					},
				},
			},
		},
		{
			// RestartPolicy deleted
			Old: &TaskGroup{
				RestartPolicy: &RestartPolicy{
					Attempts: 1,
					Interval: 1 * time.Second,
					Delay:    1 * time.Second,
					Mode:     "fail",
				},
			},
			New: &TaskGroup{},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeDeleted,
						Name: "RestartPolicy",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "Attempts",
								Old:  "1",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Delay",
								Old:  "1000000000",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Interval",
								Old:  "1000000000",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Mode",
								Old:  "fail",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// RestartPolicy edited
			Old: &TaskGroup{
				RestartPolicy: &RestartPolicy{
					Attempts: 1,
					Interval: 1 * time.Second,
					Delay:    1 * time.Second,
					Mode:     "fail",
				},
			},
			New: &TaskGroup{
				RestartPolicy: &RestartPolicy{
					Attempts: 2,
					Interval: 2 * time.Second,
					Delay:    2 * time.Second,
					Mode:     "delay",
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "RestartPolicy",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "Attempts",
								Old:  "1",
								New:  "2",
							},
							{
								Type: DiffTypeEdited,
								Name: "Delay",
								Old:  "1000000000",
								New:  "2000000000",
							},
							{
								Type: DiffTypeEdited,
								Name: "Interval",
								Old:  "1000000000",
								New:  "2000000000",
							},
							{
								Type: DiffTypeEdited,
								Name: "Mode",
								Old:  "fail",
								New:  "delay",
							},
						},
					},
				},
			},
		},
		{
			// RestartPolicy edited with context
			Contextual: true,
			Old: &TaskGroup{
				RestartPolicy: &RestartPolicy{
					Attempts: 1,
					Interval: 1 * time.Second,
					Delay:    1 * time.Second,
					Mode:     "fail",
				},
			},
			New: &TaskGroup{
				RestartPolicy: &RestartPolicy{
					Attempts: 2,
					Interval: 2 * time.Second,
					Delay:    1 * time.Second,
					Mode:     "fail",
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "RestartPolicy",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "Attempts",
								Old:  "1",
								New:  "2",
							},
							{
								Type: DiffTypeNone,
								Name: "Delay",
								Old:  "1000000000",
								New:  "1000000000",
							},
							{
								Type: DiffTypeEdited,
								Name: "Interval",
								Old:  "1000000000",
								New:  "2000000000",
							},
							{
								Type: DiffTypeNone,
								Name: "Mode",
								Old:  "fail",
								New:  "fail",
							},
						},
					},
				},
			},
		},
		{
			// Tasks edited
			Old: &TaskGroup{
				Tasks: []*Task{
					{
						Name:   "foo",
						Driver: "docker",
					},
					{
						Name:   "bar",
						Driver: "docker",
					},
					{
						Name:   "baz",
						Driver: "docker",
					},
				},
			},
			New: &TaskGroup{
				Tasks: []*Task{
					{
						Name:   "bar",
						Driver: "docker",
					},
					{
						Name:   "baz",
						Driver: "exec",
					},
					{
						Name:   "bam",
						Driver: "docker",
					},
				},
			},
			Expected: &TaskGroupDiff{
				Type: DiffTypeEdited,
				Tasks: []*TaskDiff{
					{
						Type: DiffTypeAdded,
						Name: "bam",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Driver",
								Old:  "",
								New:  "docker",
							},
							{
								Type: DiffTypeAdded,
								Name: "KillTimeout",
								Old:  "",
								New:  "0",
							},
						},
					},
					{
						Type: DiffTypeNone,
						Name: "bar",
					},
					{
						Type: DiffTypeEdited,
						Name: "baz",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "Driver",
								Old:  "docker",
								New:  "exec",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "foo",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "Driver",
								Old:  "docker",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "KillTimeout",
								Old:  "0",
								New:  "",
							},
						},
					},
				},
			},
		},
	}

	for i, c := range cases {
		actual, err := c.Old.Diff(c.New, c.Contextual)
		if c.Error && err == nil {
			t.Fatalf("case %d: expected errored")
		} else if err != nil {
			if !c.Error {
				t.Fatalf("case %d: errored %#v", i+1, err)
			} else {
				continue
			}
		}

		if !reflect.DeepEqual(actual, c.Expected) {
			t.Fatalf("case %d: got:\n%#v\n want:\n%#v\n",
				i+1, actual, c.Expected)
		}
	}
}

func TestTaskDiff(t *testing.T) {
	cases := []struct {
		Old, New   *Task
		Expected   *TaskDiff
		Error      bool
		Contextual bool
	}{
		{
			Old: nil,
			New: nil,
			Expected: &TaskDiff{
				Type: DiffTypeNone,
			},
		},
		{
			// Primitive only that has different names
			Old: &Task{
				Name: "foo",
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			New: &Task{
				Name: "bar",
				Meta: map[string]string{
					"foo": "bar",
				},
			},
			Error: true,
		},
		{
			// Primitive only that is the same
			Old: &Task{
				Name:   "foo",
				Driver: "exec",
				User:   "foo",
				Env: map[string]string{
					"FOO": "bar",
				},
				Meta: map[string]string{
					"foo": "bar",
				},
				KillTimeout: 1 * time.Second,
			},
			New: &Task{
				Name:   "foo",
				Driver: "exec",
				User:   "foo",
				Env: map[string]string{
					"FOO": "bar",
				},
				Meta: map[string]string{
					"foo": "bar",
				},
				KillTimeout: 1 * time.Second,
			},
			Expected: &TaskDiff{
				Type: DiffTypeNone,
				Name: "foo",
			},
		},
		{
			// Primitive only that has diffs
			Old: &Task{
				Name:   "foo",
				Driver: "exec",
				User:   "foo",
				Env: map[string]string{
					"FOO": "bar",
				},
				Meta: map[string]string{
					"foo": "bar",
				},
				KillTimeout: 1 * time.Second,
			},
			New: &Task{
				Name:   "foo",
				Driver: "docker",
				User:   "bar",
				Env: map[string]string{
					"FOO": "baz",
				},
				Meta: map[string]string{
					"foo": "baz",
				},
				KillTimeout: 2 * time.Second,
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Name: "foo",
				Fields: []*FieldDiff{
					{
						Type: DiffTypeEdited,
						Name: "Driver",
						Old:  "exec",
						New:  "docker",
					},
					{
						Type: DiffTypeEdited,
						Name: "Env[FOO]",
						Old:  "bar",
						New:  "baz",
					},
					{
						Type: DiffTypeEdited,
						Name: "KillTimeout",
						Old:  "1000000000",
						New:  "2000000000",
					},
					{
						Type: DiffTypeEdited,
						Name: "Meta[foo]",
						Old:  "bar",
						New:  "baz",
					},
					{
						Type: DiffTypeEdited,
						Name: "User",
						Old:  "foo",
						New:  "bar",
					},
				},
			},
		},
		{
			// Map diff
			Old: &Task{
				Meta: map[string]string{
					"foo": "foo",
					"bar": "bar",
				},
				Env: map[string]string{
					"foo": "foo",
					"bar": "bar",
				},
			},
			New: &Task{
				Meta: map[string]string{
					"bar": "bar",
					"baz": "baz",
				},
				Env: map[string]string{
					"bar": "bar",
					"baz": "baz",
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Fields: []*FieldDiff{
					{
						Type: DiffTypeAdded,
						Name: "Env[baz]",
						Old:  "",
						New:  "baz",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Env[foo]",
						Old:  "foo",
						New:  "",
					},
					{
						Type: DiffTypeAdded,
						Name: "Meta[baz]",
						Old:  "",
						New:  "baz",
					},
					{
						Type: DiffTypeDeleted,
						Name: "Meta[foo]",
						Old:  "foo",
						New:  "",
					},
				},
			},
		},
		{
			// Constraints edited
			Old: &Task{
				Constraints: []*Constraint{
					{
						LTarget: "foo",
						RTarget: "foo",
						Operand: "foo",
						str:     "foo",
					},
					{
						LTarget: "bar",
						RTarget: "bar",
						Operand: "bar",
						str:     "bar",
					},
				},
			},
			New: &Task{
				Constraints: []*Constraint{
					{
						LTarget: "foo",
						RTarget: "foo",
						Operand: "foo",
						str:     "foo",
					},
					{
						LTarget: "baz",
						RTarget: "baz",
						Operand: "baz",
						str:     "baz",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Constraint",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "LTarget",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "Operand",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "RTarget",
								Old:  "",
								New:  "baz",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "Constraint",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "LTarget",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "Operand",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "RTarget",
								Old:  "bar",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// LogConfig added
			Old: &Task{},
			New: &Task{
				LogConfig: &LogConfig{
					MaxFiles:      1,
					MaxFileSizeMB: 10,
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "LogConfig",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "MaxFileSizeMB",
								Old:  "",
								New:  "10",
							},
							{
								Type: DiffTypeAdded,
								Name: "MaxFiles",
								Old:  "",
								New:  "1",
							},
						},
					},
				},
			},
		},
		{
			// LogConfig deleted
			Old: &Task{
				LogConfig: &LogConfig{
					MaxFiles:      1,
					MaxFileSizeMB: 10,
				},
			},
			New: &Task{},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeDeleted,
						Name: "LogConfig",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "MaxFileSizeMB",
								Old:  "10",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "MaxFiles",
								Old:  "1",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// LogConfig edited
			Old: &Task{
				LogConfig: &LogConfig{
					MaxFiles:      1,
					MaxFileSizeMB: 10,
				},
			},
			New: &Task{
				LogConfig: &LogConfig{
					MaxFiles:      2,
					MaxFileSizeMB: 20,
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "LogConfig",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "MaxFileSizeMB",
								Old:  "10",
								New:  "20",
							},
							{
								Type: DiffTypeEdited,
								Name: "MaxFiles",
								Old:  "1",
								New:  "2",
							},
						},
					},
				},
			},
		},
		{
			// LogConfig edited with context
			Contextual: true,
			Old: &Task{
				LogConfig: &LogConfig{
					MaxFiles:      1,
					MaxFileSizeMB: 10,
				},
			},
			New: &Task{
				LogConfig: &LogConfig{
					MaxFiles:      1,
					MaxFileSizeMB: 20,
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "LogConfig",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "MaxFileSizeMB",
								Old:  "10",
								New:  "20",
							},
							{
								Type: DiffTypeNone,
								Name: "MaxFiles",
								Old:  "1",
								New:  "1",
							},
						},
					},
				},
			},
		},
		{
			// Artifacts edited
			Old: &Task{
				Artifacts: []*TaskArtifact{
					{
						GetterSource: "foo",
						GetterOptions: map[string]string{
							"foo": "bar",
						},
						RelativeDest: "foo",
					},
					{
						GetterSource: "bar",
						GetterOptions: map[string]string{
							"bar": "baz",
						},
						RelativeDest: "bar",
					},
				},
			},
			New: &Task{
				Artifacts: []*TaskArtifact{
					{
						GetterSource: "foo",
						GetterOptions: map[string]string{
							"foo": "bar",
						},
						RelativeDest: "foo",
					},
					{
						GetterSource: "bam",
						GetterOptions: map[string]string{
							"bam": "baz",
						},
						RelativeDest: "bam",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeAdded,
						Name: "Artifact",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "GetterOptions[bam]",
								Old:  "",
								New:  "baz",
							},
							{
								Type: DiffTypeAdded,
								Name: "GetterSource",
								Old:  "",
								New:  "bam",
							},
							{
								Type: DiffTypeAdded,
								Name: "RelativeDest",
								Old:  "",
								New:  "bam",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "Artifact",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "GetterOptions[bar]",
								Old:  "baz",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "GetterSource",
								Old:  "bar",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "RelativeDest",
								Old:  "bar",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Resources edited (no networks)
			Old: &Task{
				Resources: &Resources{
					CPU:      100,
					MemoryMB: 100,
					DiskMB:   100,
					IOPS:     100,
				},
			},
			New: &Task{
				Resources: &Resources{
					CPU:      200,
					MemoryMB: 200,
					DiskMB:   200,
					IOPS:     200,
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Resources",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "CPU",
								Old:  "100",
								New:  "200",
							},
							{
								Type: DiffTypeEdited,
								Name: "DiskMB",
								Old:  "100",
								New:  "200",
							},
							{
								Type: DiffTypeEdited,
								Name: "IOPS",
								Old:  "100",
								New:  "200",
							},
							{
								Type: DiffTypeEdited,
								Name: "MemoryMB",
								Old:  "100",
								New:  "200",
							},
						},
					},
				},
			},
		},
		{
			// Resources edited (no networks) with context
			Contextual: true,
			Old: &Task{
				Resources: &Resources{
					CPU:      100,
					MemoryMB: 100,
					DiskMB:   100,
					IOPS:     100,
				},
			},
			New: &Task{
				Resources: &Resources{
					CPU:      200,
					MemoryMB: 100,
					DiskMB:   200,
					IOPS:     100,
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Resources",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "CPU",
								Old:  "100",
								New:  "200",
							},
							{
								Type: DiffTypeEdited,
								Name: "DiskMB",
								Old:  "100",
								New:  "200",
							},
							{
								Type: DiffTypeNone,
								Name: "IOPS",
								Old:  "100",
								New:  "100",
							},
							{
								Type: DiffTypeNone,
								Name: "MemoryMB",
								Old:  "100",
								New:  "100",
							},
						},
					},
				},
			},
		},
		{
			// Network Resources edited
			Old: &Task{
				Resources: &Resources{
					Networks: []*NetworkResource{
						{
							Device: "foo",
							CIDR:   "foo",
							IP:     "foo",
							MBits:  100,
							ReservedPorts: []Port{
								{
									Label: "foo",
									Value: 80,
								},
							},
							DynamicPorts: []Port{
								{
									Label: "bar",
								},
							},
						},
					},
				},
			},
			New: &Task{
				Resources: &Resources{
					Networks: []*NetworkResource{
						{
							Device: "bar",
							CIDR:   "bar",
							IP:     "bar",
							MBits:  200,
							ReservedPorts: []Port{
								{
									Label: "foo",
									Value: 81,
								},
							},
							DynamicPorts: []Port{
								{
									Label: "baz",
								},
							},
						},
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Resources",
						Objects: []*ObjectDiff{
							{
								Type: DiffTypeAdded,
								Name: "Network",
								Fields: []*FieldDiff{
									{
										Type: DiffTypeAdded,
										Name: "MBits",
										Old:  "",
										New:  "200",
									},
								},
								Objects: []*ObjectDiff{
									{
										Type: DiffTypeAdded,
										Name: "Static Port",
										Fields: []*FieldDiff{
											{
												Type: DiffTypeAdded,
												Name: "Label",
												Old:  "",
												New:  "foo",
											},
											{
												Type: DiffTypeAdded,
												Name: "Value",
												Old:  "",
												New:  "81",
											},
										},
									},
									{
										Type: DiffTypeAdded,
										Name: "Dynamic Port",
										Fields: []*FieldDiff{
											{
												Type: DiffTypeAdded,
												Name: "Label",
												Old:  "",
												New:  "baz",
											},
										},
									},
								},
							},
							{
								Type: DiffTypeDeleted,
								Name: "Network",
								Fields: []*FieldDiff{
									{
										Type: DiffTypeDeleted,
										Name: "MBits",
										Old:  "100",
										New:  "",
									},
								},
								Objects: []*ObjectDiff{
									{
										Type: DiffTypeDeleted,
										Name: "Static Port",
										Fields: []*FieldDiff{
											{
												Type: DiffTypeDeleted,
												Name: "Label",
												Old:  "foo",
												New:  "",
											},
											{
												Type: DiffTypeDeleted,
												Name: "Value",
												Old:  "80",
												New:  "",
											},
										},
									},
									{
										Type: DiffTypeDeleted,
										Name: "Dynamic Port",
										Fields: []*FieldDiff{
											{
												Type: DiffTypeDeleted,
												Name: "Label",
												Old:  "bar",
												New:  "",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// Config same
			Old: &Task{
				Config: map[string]interface{}{
					"foo": 1,
					"bar": "bar",
					"bam": []string{"a", "b"},
					"baz": map[string]int{
						"a": 1,
						"b": 2,
					},
					"boom": &Port{
						Label: "boom_port",
					},
				},
			},
			New: &Task{
				Config: map[string]interface{}{
					"foo": 1,
					"bar": "bar",
					"bam": []string{"a", "b"},
					"baz": map[string]int{
						"a": 1,
						"b": 2,
					},
					"boom": &Port{
						Label: "boom_port",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeNone,
			},
		},
		{
			// Config edited
			Old: &Task{
				Config: map[string]interface{}{
					"foo": 1,
					"bar": "baz",
					"bam": []string{"a", "b"},
					"baz": map[string]int{
						"a": 1,
						"b": 2,
					},
					"boom": &Port{
						Label: "boom_port",
					},
				},
			},
			New: &Task{
				Config: map[string]interface{}{
					"foo": 2,
					"bar": "baz",
					"bam": []string{"a", "c", "d"},
					"baz": map[string]int{
						"b": 3,
						"c": 4,
					},
					"boom": &Port{
						Label: "boom_port2",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Config",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "bam[1]",
								Old:  "b",
								New:  "c",
							},
							{
								Type: DiffTypeAdded,
								Name: "bam[2]",
								Old:  "",
								New:  "d",
							},
							{
								Type: DiffTypeDeleted,
								Name: "baz[a]",
								Old:  "1",
								New:  "",
							},
							{
								Type: DiffTypeEdited,
								Name: "baz[b]",
								Old:  "2",
								New:  "3",
							},
							{
								Type: DiffTypeAdded,
								Name: "baz[c]",
								Old:  "",
								New:  "4",
							},
							{
								Type: DiffTypeEdited,
								Name: "boom.Label",
								Old:  "boom_port",
								New:  "boom_port2",
							},
							{
								Type: DiffTypeEdited,
								Name: "foo",
								Old:  "1",
								New:  "2",
							},
						},
					},
				},
			},
		},
		{
			// Config edited with context
			Contextual: true,
			Old: &Task{
				Config: map[string]interface{}{
					"foo": 1,
					"bar": "baz",
					"bam": []string{"a", "b"},
					"baz": map[string]int{
						"a": 1,
						"b": 2,
					},
					"boom": &Port{
						Label: "boom_port",
					},
				},
			},
			New: &Task{
				Config: map[string]interface{}{
					"foo": 2,
					"bar": "baz",
					"bam": []string{"a", "c", "d"},
					"baz": map[string]int{
						"a": 1,
						"b": 2,
					},
					"boom": &Port{
						Label: "boom_port",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Config",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeNone,
								Name: "bam[0]",
								Old:  "a",
								New:  "a",
							},
							{
								Type: DiffTypeEdited,
								Name: "bam[1]",
								Old:  "b",
								New:  "c",
							},
							{
								Type: DiffTypeAdded,
								Name: "bam[2]",
								Old:  "",
								New:  "d",
							},
							{
								Type: DiffTypeNone,
								Name: "bar",
								Old:  "baz",
								New:  "baz",
							},
							{
								Type: DiffTypeNone,
								Name: "baz[a]",
								Old:  "1",
								New:  "1",
							},
							{
								Type: DiffTypeNone,
								Name: "baz[b]",
								Old:  "2",
								New:  "2",
							},
							{
								Type: DiffTypeNone,
								Name: "boom.Label",
								Old:  "boom_port",
								New:  "boom_port",
							},
							{
								Type: DiffTypeNone,
								Name: "boom.Value",
								Old:  "0",
								New:  "0",
							},
							{
								Type: DiffTypeEdited,
								Name: "foo",
								Old:  "1",
								New:  "2",
							},
						},
					},
				},
			},
		},
		{
			// Services edited (no checks)
			Old: &Task{
				Services: []*Service{
					{
						Name:      "foo",
						PortLabel: "foo",
					},
					{
						Name:      "bar",
						PortLabel: "bar",
					},
					{
						Name:      "baz",
						PortLabel: "baz",
					},
				},
			},
			New: &Task{
				Services: []*Service{
					{
						Name:      "bar",
						PortLabel: "bar",
					},
					{
						Name:      "baz",
						PortLabel: "baz2",
					},
					{
						Name:      "bam",
						PortLabel: "bam",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Service",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeEdited,
								Name: "PortLabel",
								Old:  "baz",
								New:  "baz2",
							},
						},
					},
					{
						Type: DiffTypeAdded,
						Name: "Service",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeAdded,
								Name: "Name",
								Old:  "",
								New:  "bam",
							},
							{
								Type: DiffTypeAdded,
								Name: "PortLabel",
								Old:  "",
								New:  "bam",
							},
						},
					},
					{
						Type: DiffTypeDeleted,
						Name: "Service",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeDeleted,
								Name: "Name",
								Old:  "foo",
								New:  "",
							},
							{
								Type: DiffTypeDeleted,
								Name: "PortLabel",
								Old:  "foo",
								New:  "",
							},
						},
					},
				},
			},
		},
		{
			// Services edited (no checks) with context
			Contextual: true,
			Old: &Task{
				Services: []*Service{
					{
						Name:      "foo",
						PortLabel: "foo",
					},
				},
			},
			New: &Task{
				Services: []*Service{
					{
						Name:      "foo",
						PortLabel: "bar",
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Service",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeNone,
								Name: "Name",
								Old:  "foo",
								New:  "foo",
							},
							{
								Type: DiffTypeEdited,
								Name: "PortLabel",
								Old:  "foo",
								New:  "bar",
							},
						},
					},
				},
			},
		},
		{
			// Service Checks edited
			Old: &Task{
				Services: []*Service{
					{
						Name: "foo",
						Checks: []*ServiceCheck{
							{
								Name:     "foo",
								Type:     "http",
								Command:  "foo",
								Args:     []string{"foo"},
								Path:     "foo",
								Protocol: "http",
								Interval: 1 * time.Second,
								Timeout:  1 * time.Second,
							},
							{
								Name:     "bar",
								Type:     "http",
								Command:  "foo",
								Args:     []string{"foo"},
								Path:     "foo",
								Protocol: "http",
								Interval: 1 * time.Second,
								Timeout:  1 * time.Second,
							},
							{
								Name:     "baz",
								Type:     "http",
								Command:  "foo",
								Args:     []string{"foo"},
								Path:     "foo",
								Protocol: "http",
								Interval: 1 * time.Second,
								Timeout:  1 * time.Second,
							},
						},
					},
				},
			},
			New: &Task{
				Services: []*Service{
					{
						Name: "foo",
						Checks: []*ServiceCheck{
							{
								Name:     "bar",
								Type:     "http",
								Command:  "foo",
								Args:     []string{"foo"},
								Path:     "foo",
								Protocol: "http",
								Interval: 1 * time.Second,
								Timeout:  1 * time.Second,
							},
							{
								Name:     "baz",
								Type:     "tcp",
								Command:  "foo",
								Args:     []string{"foo"},
								Path:     "foo",
								Protocol: "http",
								Interval: 1 * time.Second,
								Timeout:  1 * time.Second,
							},
							{
								Name:     "bam",
								Type:     "http",
								Command:  "foo",
								Args:     []string{"foo"},
								Path:     "foo",
								Protocol: "http",
								Interval: 1 * time.Second,
								Timeout:  1 * time.Second,
							},
						},
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Service",
						Objects: []*ObjectDiff{
							{
								Type: DiffTypeEdited,
								Name: "Check",
								Fields: []*FieldDiff{
									{
										Type: DiffTypeEdited,
										Name: "Type",
										Old:  "http",
										New:  "tcp",
									},
								},
							},
							{
								Type: DiffTypeAdded,
								Name: "Check",
								Fields: []*FieldDiff{
									{
										Type: DiffTypeAdded,
										Name: "Command",
										Old:  "",
										New:  "foo",
									},
									{
										Type: DiffTypeAdded,
										Name: "Interval",
										Old:  "",
										New:  "1000000000",
									},
									{
										Type: DiffTypeAdded,
										Name: "Name",
										Old:  "",
										New:  "bam",
									},
									{
										Type: DiffTypeAdded,
										Name: "Path",
										Old:  "",
										New:  "foo",
									},
									{
										Type: DiffTypeAdded,
										Name: "Protocol",
										Old:  "",
										New:  "http",
									},
									{
										Type: DiffTypeAdded,
										Name: "Timeout",
										Old:  "",
										New:  "1000000000",
									},
									{
										Type: DiffTypeAdded,
										Name: "Type",
										Old:  "",
										New:  "http",
									},
								},
							},
							{
								Type: DiffTypeDeleted,
								Name: "Check",
								Fields: []*FieldDiff{
									{
										Type: DiffTypeDeleted,
										Name: "Command",
										Old:  "foo",
										New:  "",
									},
									{
										Type: DiffTypeDeleted,
										Name: "Interval",
										Old:  "1000000000",
										New:  "",
									},
									{
										Type: DiffTypeDeleted,
										Name: "Name",
										Old:  "foo",
										New:  "",
									},
									{
										Type: DiffTypeDeleted,
										Name: "Path",
										Old:  "foo",
										New:  "",
									},
									{
										Type: DiffTypeDeleted,
										Name: "Protocol",
										Old:  "http",
										New:  "",
									},
									{
										Type: DiffTypeDeleted,
										Name: "Timeout",
										Old:  "1000000000",
										New:  "",
									},
									{
										Type: DiffTypeDeleted,
										Name: "Type",
										Old:  "http",
										New:  "",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// Service Checks edited with context
			Contextual: true,
			Old: &Task{
				Services: []*Service{
					{
						Name: "foo",
						Checks: []*ServiceCheck{
							{
								Name:          "foo",
								Type:          "http",
								Command:       "foo",
								Args:          []string{"foo"},
								Path:          "foo",
								Protocol:      "http",
								Interval:      1 * time.Second,
								Timeout:       1 * time.Second,
								InitialStatus: "critical",
							},
						},
					},
				},
			},
			New: &Task{
				Services: []*Service{
					{
						Name: "foo",
						Checks: []*ServiceCheck{
							{
								Name:          "foo",
								Type:          "tcp",
								Command:       "foo",
								Args:          []string{"foo"},
								Path:          "foo",
								Protocol:      "http",
								Interval:      1 * time.Second,
								Timeout:       1 * time.Second,
								InitialStatus: "passing",
							},
						},
					},
				},
			},
			Expected: &TaskDiff{
				Type: DiffTypeEdited,
				Objects: []*ObjectDiff{
					{
						Type: DiffTypeEdited,
						Name: "Service",
						Fields: []*FieldDiff{
							{
								Type: DiffTypeNone,
								Name: "Name",
								Old:  "foo",
								New:  "foo",
							},
							{
								Type: DiffTypeNone,
								Name: "PortLabel",
								Old:  "",
								New:  "",
							},
						},
						Objects: []*ObjectDiff{
							{
								Type: DiffTypeEdited,
								Name: "Check",
								Fields: []*FieldDiff{
									{
										Type: DiffTypeNone,
										Name: "Command",
										Old:  "foo",
										New:  "foo",
									},
									{
										Type: DiffTypeEdited,
										Name: "InitialStatus",
										Old:  "critical",
										New:  "passing",
									},
									{
										Type: DiffTypeNone,
										Name: "Interval",
										Old:  "1000000000",
										New:  "1000000000",
									},
									{
										Type: DiffTypeNone,
										Name: "Name",
										Old:  "foo",
										New:  "foo",
									},
									{
										Type: DiffTypeNone,
										Name: "Path",
										Old:  "foo",
										New:  "foo",
									},
									{
										Type: DiffTypeNone,
										Name: "PortLabel",
										Old:  "",
										New:  "",
									},
									{
										Type: DiffTypeNone,
										Name: "Protocol",
										Old:  "http",
										New:  "http",
									},
									{
										Type: DiffTypeNone,
										Name: "Timeout",
										Old:  "1000000000",
										New:  "1000000000",
									},
									{
										Type: DiffTypeEdited,
										Name: "Type",
										Old:  "http",
										New:  "tcp",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, c := range cases {
		actual, err := c.Old.Diff(c.New, c.Contextual)
		if c.Error && err == nil {
			t.Fatalf("case %d: expected errored", i+1)
		} else if err != nil {
			if !c.Error {
				t.Fatalf("case %d: errored %#v", i+1, err)
			} else {
				continue
			}
		}

		if !reflect.DeepEqual(actual, c.Expected) {
			t.Errorf("case %d: got:\n%#v\n want:\n%#v\n",
				i+1, actual, c.Expected)
		}
	}
}
