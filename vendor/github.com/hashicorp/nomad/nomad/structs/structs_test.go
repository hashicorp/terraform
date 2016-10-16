package structs

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-multierror"
)

func TestJob_Validate(t *testing.T) {
	j := &Job{}
	err := j.Validate()
	mErr := err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "job region") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[1].Error(), "job ID") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[2].Error(), "job name") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[3].Error(), "job type") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[4].Error(), "priority") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[5].Error(), "datacenters") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[6].Error(), "task groups") {
		t.Fatalf("err: %s", err)
	}

	j = &Job{
		Type: JobTypeService,
		Periodic: &PeriodicConfig{
			Enabled: true,
		},
	}
	err = j.Validate()
	mErr = err.(*multierror.Error)
	if !strings.Contains(mErr.Error(), "Periodic") {
		t.Fatalf("err: %s", err)
	}

	j = &Job{
		Region:      "global",
		ID:          GenerateUUID(),
		Name:        "my-job",
		Type:        JobTypeService,
		Priority:    50,
		Datacenters: []string{"dc1"},
		TaskGroups: []*TaskGroup{
			&TaskGroup{
				Name: "web",
				RestartPolicy: &RestartPolicy{
					Interval: 5 * time.Minute,
					Delay:    10 * time.Second,
					Attempts: 10,
				},
			},
			&TaskGroup{
				Name: "web",
				RestartPolicy: &RestartPolicy{
					Interval: 5 * time.Minute,
					Delay:    10 * time.Second,
					Attempts: 10,
				},
			},
			&TaskGroup{
				RestartPolicy: &RestartPolicy{
					Interval: 5 * time.Minute,
					Delay:    10 * time.Second,
					Attempts: 10,
				},
			},
		},
	}
	err = j.Validate()
	mErr = err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "2 redefines 'web' from group 1") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[1].Error(), "group 3 missing name") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[2].Error(), "Task group web validation failed") {
		t.Fatalf("err: %s", err)
	}
}

func testJob() *Job {
	return &Job{
		Region:      "global",
		ID:          GenerateUUID(),
		Name:        "my-job",
		Type:        JobTypeService,
		Priority:    50,
		AllAtOnce:   false,
		Datacenters: []string{"dc1"},
		Constraints: []*Constraint{
			&Constraint{
				LTarget: "$attr.kernel.name",
				RTarget: "linux",
				Operand: "=",
			},
		},
		Periodic: &PeriodicConfig{
			Enabled: false,
		},
		TaskGroups: []*TaskGroup{
			&TaskGroup{
				Name:  "web",
				Count: 10,
				RestartPolicy: &RestartPolicy{
					Mode:     RestartPolicyModeFail,
					Attempts: 3,
					Interval: 10 * time.Minute,
					Delay:    1 * time.Minute,
				},
				Tasks: []*Task{
					&Task{
						Name:   "web",
						Driver: "exec",
						Config: map[string]interface{}{
							"command": "/bin/date",
						},
						Env: map[string]string{
							"FOO": "bar",
						},
						Artifacts: []*TaskArtifact{
							{
								GetterSource: "http://foo.com",
							},
						},
						Services: []*Service{
							{
								Name:      "${TASK}-frontend",
								PortLabel: "http",
							},
						},
						Resources: &Resources{
							CPU:      500,
							MemoryMB: 256,
							DiskMB:   20,
							Networks: []*NetworkResource{
								&NetworkResource{
									MBits:        50,
									DynamicPorts: []Port{{Label: "http"}},
								},
							},
						},
						LogConfig: &LogConfig{
							MaxFiles:      10,
							MaxFileSizeMB: 1,
						},
					},
				},
				Meta: map[string]string{
					"elb_check_type":     "http",
					"elb_check_interval": "30s",
					"elb_check_min":      "3",
				},
			},
		},
		Meta: map[string]string{
			"owner": "armon",
		},
	}
}

func TestJob_Copy(t *testing.T) {
	j := testJob()
	c := j.Copy()
	if !reflect.DeepEqual(j, c) {
		t.Fatalf("Copy() returned an unequal Job; got %#v; want %#v", c, j)
	}
}

func TestJob_IsPeriodic(t *testing.T) {
	j := &Job{
		Type: JobTypeService,
		Periodic: &PeriodicConfig{
			Enabled: true,
		},
	}
	if !j.IsPeriodic() {
		t.Fatalf("IsPeriodic() returned false on periodic job")
	}

	j = &Job{
		Type: JobTypeService,
	}
	if j.IsPeriodic() {
		t.Fatalf("IsPeriodic() returned true on non-periodic job")
	}
}

func TestJob_SystemJob_Validate(t *testing.T) {
	j := testJob()
	j.Type = JobTypeSystem
	j.Canonicalize()

	err := j.Validate()
	if err == nil || !strings.Contains(err.Error(), "exceed") {
		t.Fatalf("expect error due to count")
	}

	j.TaskGroups[0].Count = 0
	if err := j.Validate(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	j.TaskGroups[0].Count = 1
	if err := j.Validate(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestJob_VaultPolicies(t *testing.T) {
	j0 := &Job{}
	e0 := make(map[string]map[string][]string, 0)

	j1 := &Job{
		TaskGroups: []*TaskGroup{
			&TaskGroup{
				Name: "foo",
				Tasks: []*Task{
					&Task{
						Name: "t1",
					},
					&Task{
						Name: "t2",
						Vault: &Vault{
							Policies: []string{
								"p1",
								"p2",
							},
						},
					},
				},
			},
			&TaskGroup{
				Name: "bar",
				Tasks: []*Task{
					&Task{
						Name: "t3",
						Vault: &Vault{
							Policies: []string{
								"p3",
								"p4",
							},
						},
					},
					&Task{
						Name: "t4",
						Vault: &Vault{
							Policies: []string{
								"p5",
							},
						},
					},
				},
			},
		},
	}

	e1 := map[string]map[string][]string{
		"foo": map[string][]string{
			"t2": []string{"p1", "p2"},
		},
		"bar": map[string][]string{
			"t3": []string{"p3", "p4"},
			"t4": []string{"p5"},
		},
	}

	cases := []struct {
		Job      *Job
		Expected map[string]map[string][]string
	}{
		{
			Job:      j0,
			Expected: e0,
		},
		{
			Job:      j1,
			Expected: e1,
		},
	}

	for i, c := range cases {
		got := c.Job.VaultPolicies()
		if !reflect.DeepEqual(got, c.Expected) {
			t.Fatalf("case %d: got %#v; want %#v", i+1, got, c.Expected)
		}
	}
}

func TestTaskGroup_Validate(t *testing.T) {
	tg := &TaskGroup{
		Count: -1,
		RestartPolicy: &RestartPolicy{
			Interval: 5 * time.Minute,
			Delay:    10 * time.Second,
			Attempts: 10,
			Mode:     RestartPolicyModeDelay,
		},
	}
	err := tg.Validate()
	mErr := err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "group name") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[1].Error(), "count can't be negative") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[2].Error(), "Missing tasks") {
		t.Fatalf("err: %s", err)
	}

	tg = &TaskGroup{
		Name:  "web",
		Count: 1,
		Tasks: []*Task{
			&Task{Name: "web"},
			&Task{Name: "web"},
			&Task{},
		},
		RestartPolicy: &RestartPolicy{
			Interval: 5 * time.Minute,
			Delay:    10 * time.Second,
			Attempts: 10,
			Mode:     RestartPolicyModeDelay,
		},
	}

	err = tg.Validate()
	mErr = err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "2 redefines 'web' from task 1") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[1].Error(), "Task 3 missing name") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[2].Error(), "Task web validation failed") {
		t.Fatalf("err: %s", err)
	}
}

func TestTask_Validate(t *testing.T) {
	task := &Task{}
	err := task.Validate()
	mErr := err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "task name") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[1].Error(), "task driver") {
		t.Fatalf("err: %s", err)
	}
	if !strings.Contains(mErr.Errors[2].Error(), "task resources") {
		t.Fatalf("err: %s", err)
	}

	task = &Task{Name: "web/foo"}
	err = task.Validate()
	mErr = err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "slashes") {
		t.Fatalf("err: %s", err)
	}

	task = &Task{
		Name:   "web",
		Driver: "docker",
		Resources: &Resources{
			CPU:      100,
			DiskMB:   200,
			MemoryMB: 100,
			IOPS:     10,
		},
		LogConfig: DefaultLogConfig(),
	}
	err = task.Validate()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestTask_Validate_Services(t *testing.T) {
	s1 := &Service{
		Name:      "service-name",
		PortLabel: "bar",
		Checks: []*ServiceCheck{
			{
				Name:     "check-name",
				Type:     ServiceCheckTCP,
				Interval: 0 * time.Second,
			},
			{
				Name:    "check-name",
				Type:    ServiceCheckTCP,
				Timeout: 2 * time.Second,
			},
		},
	}

	s2 := &Service{
		Name: "service-name",
	}

	task := &Task{
		Name:   "web",
		Driver: "docker",
		Resources: &Resources{
			CPU:      100,
			DiskMB:   200,
			MemoryMB: 100,
			IOPS:     10,
		},
		Services: []*Service{s1, s2},
	}
	err := task.Validate()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "referenced by services service-name does not exist") {
		t.Fatalf("err: %s", err)
	}

	if !strings.Contains(err.Error(), "service \"service-name\" is duplicate") {
		t.Fatalf("err: %v", err)
	}

	if !strings.Contains(err.Error(), "check \"check-name\" is duplicate") {
		t.Fatalf("err: %v", err)
	}

	if !strings.Contains(err.Error(), "interval (0s) can not be lower") {
		t.Fatalf("err: %v", err)
	}
}

func TestTask_Validate_Service_Check(t *testing.T) {

	check1 := ServiceCheck{
		Name:     "check-name",
		Type:     ServiceCheckTCP,
		Interval: 10 * time.Second,
		Timeout:  2 * time.Second,
	}

	err := check1.validate()
	if err != nil {
		t.Fatal("err: %v", err)
	}

	check1.InitialStatus = "foo"
	err = check1.validate()
	if err == nil {
		t.Fatal("Expected an error")
	}

	if !strings.Contains(err.Error(), "invalid initial check state (foo)") {
		t.Fatalf("err: %v", err)
	}

	check1.InitialStatus = api.HealthCritical
	err = check1.validate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	check1.InitialStatus = api.HealthPassing
	err = check1.validate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	check1.InitialStatus = ""
	err = check1.validate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestTask_Validate_LogConfig(t *testing.T) {
	task := &Task{
		LogConfig: DefaultLogConfig(),
		Resources: &Resources{
			DiskMB: 1,
		},
	}

	err := task.Validate()
	mErr := err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[3].Error(), "log storage") {
		t.Fatalf("err: %s", err)
	}
}

func TestConstraint_Validate(t *testing.T) {
	c := &Constraint{}
	err := c.Validate()
	mErr := err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "Missing constraint operand") {
		t.Fatalf("err: %s", err)
	}

	c = &Constraint{
		LTarget: "$attr.kernel.name",
		RTarget: "linux",
		Operand: "=",
	}
	err = c.Validate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Perform additional regexp validation
	c.Operand = ConstraintRegex
	c.RTarget = "(foo"
	err = c.Validate()
	mErr = err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "missing closing") {
		t.Fatalf("err: %s", err)
	}

	// Perform version validation
	c.Operand = ConstraintVersion
	c.RTarget = "~> foo"
	err = c.Validate()
	mErr = err.(*multierror.Error)
	if !strings.Contains(mErr.Errors[0].Error(), "Malformed constraint") {
		t.Fatalf("err: %s", err)
	}
}

func TestResource_NetIndex(t *testing.T) {
	r := &Resources{
		Networks: []*NetworkResource{
			&NetworkResource{Device: "eth0"},
			&NetworkResource{Device: "lo0"},
			&NetworkResource{Device: ""},
		},
	}
	if idx := r.NetIndex(&NetworkResource{Device: "eth0"}); idx != 0 {
		t.Fatalf("Bad: %d", idx)
	}
	if idx := r.NetIndex(&NetworkResource{Device: "lo0"}); idx != 1 {
		t.Fatalf("Bad: %d", idx)
	}
	if idx := r.NetIndex(&NetworkResource{Device: "eth1"}); idx != -1 {
		t.Fatalf("Bad: %d", idx)
	}
}

func TestResource_Superset(t *testing.T) {
	r1 := &Resources{
		CPU:      2000,
		MemoryMB: 2048,
		DiskMB:   10000,
		IOPS:     100,
	}
	r2 := &Resources{
		CPU:      2000,
		MemoryMB: 1024,
		DiskMB:   5000,
		IOPS:     50,
	}

	if s, _ := r1.Superset(r1); !s {
		t.Fatalf("bad")
	}
	if s, _ := r1.Superset(r2); !s {
		t.Fatalf("bad")
	}
	if s, _ := r2.Superset(r1); s {
		t.Fatalf("bad")
	}
	if s, _ := r2.Superset(r2); !s {
		t.Fatalf("bad")
	}
}

func TestResource_Add(t *testing.T) {
	r1 := &Resources{
		CPU:      2000,
		MemoryMB: 2048,
		DiskMB:   10000,
		IOPS:     100,
		Networks: []*NetworkResource{
			&NetworkResource{
				CIDR:          "10.0.0.0/8",
				MBits:         100,
				ReservedPorts: []Port{{"ssh", 22}},
			},
		},
	}
	r2 := &Resources{
		CPU:      2000,
		MemoryMB: 1024,
		DiskMB:   5000,
		IOPS:     50,
		Networks: []*NetworkResource{
			&NetworkResource{
				IP:            "10.0.0.1",
				MBits:         50,
				ReservedPorts: []Port{{"web", 80}},
			},
		},
	}

	err := r1.Add(r2)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	expect := &Resources{
		CPU:      3000,
		MemoryMB: 3072,
		DiskMB:   15000,
		IOPS:     150,
		Networks: []*NetworkResource{
			&NetworkResource{
				CIDR:          "10.0.0.0/8",
				MBits:         150,
				ReservedPorts: []Port{{"ssh", 22}, {"web", 80}},
			},
		},
	}

	if !reflect.DeepEqual(expect.Networks, r1.Networks) {
		t.Fatalf("bad: %#v %#v", expect, r1)
	}
}

func TestResource_Add_Network(t *testing.T) {
	r1 := &Resources{}
	r2 := &Resources{
		Networks: []*NetworkResource{
			&NetworkResource{
				MBits:        50,
				DynamicPorts: []Port{{"http", 0}, {"https", 0}},
			},
		},
	}
	r3 := &Resources{
		Networks: []*NetworkResource{
			&NetworkResource{
				MBits:        25,
				DynamicPorts: []Port{{"admin", 0}},
			},
		},
	}

	err := r1.Add(r2)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	err = r1.Add(r3)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	expect := &Resources{
		Networks: []*NetworkResource{
			&NetworkResource{
				MBits:        75,
				DynamicPorts: []Port{{"http", 0}, {"https", 0}, {"admin", 0}},
			},
		},
	}

	if !reflect.DeepEqual(expect.Networks, r1.Networks) {
		t.Fatalf("bad: %#v %#v", expect.Networks[0], r1.Networks[0])
	}
}

func TestEncodeDecode(t *testing.T) {
	type FooRequest struct {
		Foo string
		Bar int
		Baz bool
	}
	arg := &FooRequest{
		Foo: "test",
		Bar: 42,
		Baz: true,
	}
	buf, err := Encode(1, arg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var out FooRequest
	err = Decode(buf[1:], &out)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(arg, &out) {
		t.Fatalf("bad: %#v %#v", arg, out)
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	job := testJob()

	for i := 0; i < b.N; i++ {
		buf, err := Encode(1, job)
		if err != nil {
			b.Fatalf("err: %v", err)
		}

		var out Job
		err = Decode(buf[1:], &out)
		if err != nil {
			b.Fatalf("err: %v", err)
		}
	}
}

func TestInvalidServiceCheck(t *testing.T) {
	s := Service{
		Name:      "service-name",
		PortLabel: "bar",
		Checks: []*ServiceCheck{
			{
				Name: "check-name",
				Type: "lol",
			},
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatalf("Service should be invalid (invalid type)")
	}

	s = Service{
		Name:      "service.name",
		PortLabel: "bar",
	}
	if err := s.Validate(); err == nil {
		t.Fatalf("Service should be invalid (contains a dot): %v", err)
	}

	s = Service{
		Name:      "-my-service",
		PortLabel: "bar",
	}
	if err := s.Validate(); err == nil {
		t.Fatalf("Service should be invalid (begins with a hyphen): %v", err)
	}

	s = Service{
		Name:      "abcdef0123456789-abcdef0123456789-abcdef0123456789-abcdef0123456",
		PortLabel: "bar",
	}
	if err := s.Validate(); err == nil {
		t.Fatalf("Service should be invalid (too long): %v", err)
	}

	s = Service{
		Name: "service-name",
		Checks: []*ServiceCheck{
			{
				Name:     "check-tcp",
				Type:     ServiceCheckTCP,
				Interval: 5 * time.Second,
				Timeout:  2 * time.Second,
			},
			{
				Name:     "check-http",
				Type:     ServiceCheckHTTP,
				Path:     "/foo",
				Interval: 5 * time.Second,
				Timeout:  2 * time.Second,
			},
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatalf("service should be invalid (tcp/http checks with no port): %v", err)
	}

	s = Service{
		Name: "service-name",
		Checks: []*ServiceCheck{
			{
				Name:     "check-script",
				Type:     ServiceCheckScript,
				Command:  "/bin/date",
				Interval: 5 * time.Second,
				Timeout:  2 * time.Second,
			},
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("un-expected error: %v", err)
	}
}

func TestDistinctCheckID(t *testing.T) {
	c1 := ServiceCheck{
		Name:     "web-health",
		Type:     "http",
		Path:     "/health",
		Interval: 2 * time.Second,
		Timeout:  3 * time.Second,
	}
	c2 := ServiceCheck{
		Name:     "web-health",
		Type:     "http",
		Path:     "/health1",
		Interval: 2 * time.Second,
		Timeout:  3 * time.Second,
	}

	c3 := ServiceCheck{
		Name:     "web-health",
		Type:     "http",
		Path:     "/health",
		Interval: 4 * time.Second,
		Timeout:  3 * time.Second,
	}
	serviceID := "123"
	c1Hash := c1.Hash(serviceID)
	c2Hash := c2.Hash(serviceID)
	c3Hash := c3.Hash(serviceID)

	if c1Hash == c2Hash || c1Hash == c3Hash || c3Hash == c2Hash {
		t.Fatalf("Checks need to be uniq c1: %s, c2: %s, c3: %s", c1Hash, c2Hash, c3Hash)
	}

}

func TestService_Canonicalize(t *testing.T) {
	job := "example"
	taskGroup := "cache"
	task := "redis"

	s := Service{
		Name: "${TASK}-db",
	}

	s.Canonicalize(job, taskGroup, task)
	if s.Name != "redis-db" {
		t.Fatalf("Expected name: %v, Actual: %v", "redis-db", s.Name)
	}

	s.Name = "db"
	s.Canonicalize(job, taskGroup, task)
	if s.Name != "db" {
		t.Fatalf("Expected name: %v, Actual: %v", "redis-db", s.Name)
	}

	s.Name = "${JOB}-${TASKGROUP}-${TASK}-db"
	s.Canonicalize(job, taskGroup, task)
	if s.Name != "example-cache-redis-db" {
		t.Fatalf("Expected name: %v, Actual: %v", "expample-cache-redis-db", s.Name)
	}

	s.Name = "${BASE}-db"
	s.Canonicalize(job, taskGroup, task)
	if s.Name != "example-cache-redis-db" {
		t.Fatalf("Expected name: %v, Actual: %v", "expample-cache-redis-db", s.Name)
	}

}

func TestJob_ExpandServiceNames(t *testing.T) {
	j := &Job{
		Name: "my-job",
		TaskGroups: []*TaskGroup{
			&TaskGroup{
				Name: "web",
				Tasks: []*Task{
					{
						Name: "frontend",
						Services: []*Service{
							{
								Name: "${BASE}-default",
							},
							{
								Name: "jmx",
							},
						},
					},
				},
			},
			&TaskGroup{
				Name: "admin",
				Tasks: []*Task{
					{
						Name: "admin-web",
					},
				},
			},
		},
	}

	j.Canonicalize()

	service1Name := j.TaskGroups[0].Tasks[0].Services[0].Name
	if service1Name != "my-job-web-frontend-default" {
		t.Fatalf("Expected Service Name: %s, Actual: %s", "my-job-web-frontend-default", service1Name)
	}

	service2Name := j.TaskGroups[0].Tasks[0].Services[1].Name
	if service2Name != "jmx" {
		t.Fatalf("Expected Service Name: %s, Actual: %s", "jmx", service2Name)
	}

}

func TestPeriodicConfig_EnabledInvalid(t *testing.T) {
	// Create a config that is enabled but with no interval specified.
	p := &PeriodicConfig{Enabled: true}
	if err := p.Validate(); err == nil {
		t.Fatal("Enabled PeriodicConfig with no spec or type shouldn't be valid")
	}

	// Create a config that is enabled, with a spec but no type specified.
	p = &PeriodicConfig{Enabled: true, Spec: "foo"}
	if err := p.Validate(); err == nil {
		t.Fatal("Enabled PeriodicConfig with no spec type shouldn't be valid")
	}

	// Create a config that is enabled, with a spec type but no spec specified.
	p = &PeriodicConfig{Enabled: true, SpecType: PeriodicSpecCron}
	if err := p.Validate(); err == nil {
		t.Fatal("Enabled PeriodicConfig with no spec shouldn't be valid")
	}
}

func TestPeriodicConfig_InvalidCron(t *testing.T) {
	specs := []string{"foo", "* *", "@foo"}
	for _, spec := range specs {
		p := &PeriodicConfig{Enabled: true, SpecType: PeriodicSpecCron, Spec: spec}
		if err := p.Validate(); err == nil {
			t.Fatal("Invalid cron spec")
		}
	}
}

func TestPeriodicConfig_ValidCron(t *testing.T) {
	specs := []string{"0 0 29 2 *", "@hourly", "0 0-15 * * *"}
	for _, spec := range specs {
		p := &PeriodicConfig{Enabled: true, SpecType: PeriodicSpecCron, Spec: spec}
		if err := p.Validate(); err != nil {
			t.Fatal("Passed valid cron")
		}
	}
}

func TestPeriodicConfig_NextCron(t *testing.T) {
	from := time.Date(2009, time.November, 10, 23, 22, 30, 0, time.UTC)
	specs := []string{"0 0 29 2 * 1980", "*/5 * * * *"}
	expected := []time.Time{time.Time{}, time.Date(2009, time.November, 10, 23, 25, 0, 0, time.UTC)}
	for i, spec := range specs {
		p := &PeriodicConfig{Enabled: true, SpecType: PeriodicSpecCron, Spec: spec}
		n := p.Next(from)
		if expected[i] != n {
			t.Fatalf("Next(%v) returned %v; want %v", from, n, expected[i])
		}
	}
}

func TestRestartPolicy_Validate(t *testing.T) {
	// Policy with acceptable restart options passes
	p := &RestartPolicy{
		Mode:     RestartPolicyModeFail,
		Attempts: 0,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Policy with ambiguous restart options fails
	p = &RestartPolicy{
		Mode:     RestartPolicyModeDelay,
		Attempts: 0,
	}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expect ambiguity error, got: %v", err)
	}

	// Bad policy mode fails
	p = &RestartPolicy{
		Mode:     "nope",
		Attempts: 1,
	}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatalf("expect mode error, got: %v", err)
	}

	// Fails when attempts*delay does not fit inside interval
	p = &RestartPolicy{
		Mode:     RestartPolicyModeDelay,
		Attempts: 3,
		Delay:    5 * time.Second,
		Interval: time.Second,
	}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "can't restart") {
		t.Fatalf("expect restart interval error, got: %v", err)
	}
}

func TestAllocation_Index(t *testing.T) {
	a1 := Allocation{Name: "example.cache[0]"}
	e1 := 0
	a2 := Allocation{Name: "ex[123]am123ple.c311ac[123]he12[1][77]"}
	e2 := 77

	if a1.Index() != e1 || a2.Index() != e2 {
		t.Fatal()
	}
}

func TestTaskArtifact_Validate_Source(t *testing.T) {
	valid := &TaskArtifact{GetterSource: "google.com"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskArtifact_Validate_Dest(t *testing.T) {
	valid := &TaskArtifact{GetterSource: "google.com"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	valid.RelativeDest = "local/"
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	valid.RelativeDest = "local/.."
	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	valid.RelativeDest = "local/../.."
	if err := valid.Validate(); err == nil {
		t.Fatalf("expected error: %v", err)
	}
}

func TestTaskArtifact_Validate_Checksum(t *testing.T) {
	cases := []struct {
		Input *TaskArtifact
		Err   bool
	}{
		{
			&TaskArtifact{
				GetterSource: "foo.com",
				GetterOptions: map[string]string{
					"checksum": "no-type",
				},
			},
			true,
		},
		{
			&TaskArtifact{
				GetterSource: "foo.com",
				GetterOptions: map[string]string{
					"checksum": "md5:toosmall",
				},
			},
			true,
		},
		{
			&TaskArtifact{
				GetterSource: "foo.com",
				GetterOptions: map[string]string{
					"checksum": "invalid:type",
				},
			},
			true,
		},
	}

	for i, tc := range cases {
		err := tc.Input.Validate()
		if (err != nil) != tc.Err {
			t.Fatalf("case %d: %v", i, err)
			continue
		}
	}
}
