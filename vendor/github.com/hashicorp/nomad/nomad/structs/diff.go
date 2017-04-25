package structs

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/nomad/helper/flatmap"
	"github.com/mitchellh/hashstructure"
)

// DiffType denotes the type of a diff object.
type DiffType string

var (
	DiffTypeNone    DiffType = "None"
	DiffTypeAdded   DiffType = "Added"
	DiffTypeDeleted DiffType = "Deleted"
	DiffTypeEdited  DiffType = "Edited"
)

func (d DiffType) Less(other DiffType) bool {
	// Edited > Added > Deleted > None
	// But we do a reverse sort
	if d == other {
		return false
	}

	if d == DiffTypeEdited {
		return true
	} else if other == DiffTypeEdited {
		return false
	} else if d == DiffTypeAdded {
		return true
	} else if other == DiffTypeAdded {
		return false
	} else if d == DiffTypeDeleted {
		return true
	} else if other == DiffTypeDeleted {
		return false
	}

	return true
}

// JobDiff contains the diff of two jobs.
type JobDiff struct {
	Type       DiffType
	ID         string
	Fields     []*FieldDiff
	Objects    []*ObjectDiff
	TaskGroups []*TaskGroupDiff
}

// Diff returns a diff of two jobs and a potential error if the Jobs are not
// diffable. If contextual diff is enabled, objects within the job will contain
// field information even if unchanged.
func (j *Job) Diff(other *Job, contextual bool) (*JobDiff, error) {
	diff := &JobDiff{Type: DiffTypeNone}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string
	filter := []string{"ID", "Status", "StatusDescription", "CreateIndex", "ModifyIndex", "JobModifyIndex"}

	// Have to treat this special since it is a struct literal, not a pointer
	var jUpdate, otherUpdate *UpdateStrategy

	if j == nil && other == nil {
		return diff, nil
	} else if j == nil {
		j = &Job{}
		otherUpdate = &other.Update
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
		diff.ID = other.ID
	} else if other == nil {
		other = &Job{}
		jUpdate = &j.Update
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(j, filter, true)
		diff.ID = j.ID
	} else {
		if j.ID != other.ID {
			return nil, fmt.Errorf("can not diff jobs with different IDs: %q and %q", j.ID, other.ID)
		}

		if !reflect.DeepEqual(j, other) {
			diff.Type = DiffTypeEdited
		}

		jUpdate = &j.Update
		otherUpdate = &other.Update
		oldPrimitiveFlat = flatmap.Flatten(j, filter, true)
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
		diff.ID = other.ID
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, false)

	// Datacenters diff
	if setDiff := stringSetDiff(j.Datacenters, other.Datacenters, "Datacenters", contextual); setDiff != nil {
		diff.Objects = append(diff.Objects, setDiff)
	}

	// Constraints diff
	conDiff := primitiveObjectSetDiff(
		interfaceSlice(j.Constraints),
		interfaceSlice(other.Constraints),
		[]string{"str"},
		"Constraint",
		contextual)
	if conDiff != nil {
		diff.Objects = append(diff.Objects, conDiff...)
	}

	// Task groups diff
	tgs, err := taskGroupDiffs(j.TaskGroups, other.TaskGroups, contextual)
	if err != nil {
		return nil, err
	}
	diff.TaskGroups = tgs

	// Update diff
	if uDiff := primitiveObjectDiff(jUpdate, otherUpdate, nil, "Update", contextual); uDiff != nil {
		diff.Objects = append(diff.Objects, uDiff)
	}

	// Periodic diff
	if pDiff := primitiveObjectDiff(j.Periodic, other.Periodic, nil, "Periodic", contextual); pDiff != nil {
		diff.Objects = append(diff.Objects, pDiff)
	}

	// ParameterizedJob diff
	if cDiff := parameterizedJobDiff(j.ParameterizedJob, other.ParameterizedJob, contextual); cDiff != nil {
		diff.Objects = append(diff.Objects, cDiff)
	}

	return diff, nil
}

func (j *JobDiff) GoString() string {
	out := fmt.Sprintf("Job %q (%s):\n", j.ID, j.Type)

	for _, f := range j.Fields {
		out += fmt.Sprintf("%#v\n", f)
	}

	for _, o := range j.Objects {
		out += fmt.Sprintf("%#v\n", o)
	}

	for _, tg := range j.TaskGroups {
		out += fmt.Sprintf("%#v\n", tg)
	}

	return out
}

// TaskGroupDiff contains the diff of two task groups.
type TaskGroupDiff struct {
	Type    DiffType
	Name    string
	Fields  []*FieldDiff
	Objects []*ObjectDiff
	Tasks   []*TaskDiff
	Updates map[string]uint64
}

// Diff returns a diff of two task groups. If contextual diff is enabled,
// objects' fields will be stored even if no diff occurred as long as one field
// changed.
func (tg *TaskGroup) Diff(other *TaskGroup, contextual bool) (*TaskGroupDiff, error) {
	diff := &TaskGroupDiff{Type: DiffTypeNone}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string
	filter := []string{"Name"}

	if tg == nil && other == nil {
		return diff, nil
	} else if tg == nil {
		tg = &TaskGroup{}
		diff.Type = DiffTypeAdded
		diff.Name = other.Name
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
	} else if other == nil {
		other = &TaskGroup{}
		diff.Type = DiffTypeDeleted
		diff.Name = tg.Name
		oldPrimitiveFlat = flatmap.Flatten(tg, filter, true)
	} else {
		if !reflect.DeepEqual(tg, other) {
			diff.Type = DiffTypeEdited
		}
		if tg.Name != other.Name {
			return nil, fmt.Errorf("can not diff task groups with different names: %q and %q", tg.Name, other.Name)
		}
		diff.Name = other.Name
		oldPrimitiveFlat = flatmap.Flatten(tg, filter, true)
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, false)

	// Constraints diff
	conDiff := primitiveObjectSetDiff(
		interfaceSlice(tg.Constraints),
		interfaceSlice(other.Constraints),
		[]string{"str"},
		"Constraint",
		contextual)
	if conDiff != nil {
		diff.Objects = append(diff.Objects, conDiff...)
	}

	// Restart policy diff
	rDiff := primitiveObjectDiff(tg.RestartPolicy, other.RestartPolicy, nil, "RestartPolicy", contextual)
	if rDiff != nil {
		diff.Objects = append(diff.Objects, rDiff)
	}

	// EphemeralDisk diff
	diskDiff := primitiveObjectDiff(tg.EphemeralDisk, other.EphemeralDisk, nil, "EphemeralDisk", contextual)
	if diskDiff != nil {
		diff.Objects = append(diff.Objects, diskDiff)
	}

	// Tasks diff
	tasks, err := taskDiffs(tg.Tasks, other.Tasks, contextual)
	if err != nil {
		return nil, err
	}
	diff.Tasks = tasks

	return diff, nil
}

func (tg *TaskGroupDiff) GoString() string {
	out := fmt.Sprintf("Group %q (%s):\n", tg.Name, tg.Type)

	if len(tg.Updates) != 0 {
		out += "Updates {\n"
		for update, count := range tg.Updates {
			out += fmt.Sprintf("%d %s\n", count, update)
		}
		out += "}\n"
	}

	for _, f := range tg.Fields {
		out += fmt.Sprintf("%#v\n", f)
	}

	for _, o := range tg.Objects {
		out += fmt.Sprintf("%#v\n", o)
	}

	for _, t := range tg.Tasks {
		out += fmt.Sprintf("%#v\n", t)
	}

	return out
}

// TaskGroupDiffs diffs two sets of task groups. If contextual diff is enabled,
// objects' fields will be stored even if no diff occurred as long as one field
// changed.
func taskGroupDiffs(old, new []*TaskGroup, contextual bool) ([]*TaskGroupDiff, error) {
	oldMap := make(map[string]*TaskGroup, len(old))
	newMap := make(map[string]*TaskGroup, len(new))
	for _, o := range old {
		oldMap[o.Name] = o
	}
	for _, n := range new {
		newMap[n.Name] = n
	}

	var diffs []*TaskGroupDiff
	for name, oldGroup := range oldMap {
		// Diff the same, deleted and edited
		diff, err := oldGroup.Diff(newMap[name], contextual)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, diff)
	}

	for name, newGroup := range newMap {
		// Diff the added
		if old, ok := oldMap[name]; !ok {
			diff, err := old.Diff(newGroup, contextual)
			if err != nil {
				return nil, err
			}
			diffs = append(diffs, diff)
		}
	}

	sort.Sort(TaskGroupDiffs(diffs))
	return diffs, nil
}

// For sorting TaskGroupDiffs
type TaskGroupDiffs []*TaskGroupDiff

func (tg TaskGroupDiffs) Len() int           { return len(tg) }
func (tg TaskGroupDiffs) Swap(i, j int)      { tg[i], tg[j] = tg[j], tg[i] }
func (tg TaskGroupDiffs) Less(i, j int) bool { return tg[i].Name < tg[j].Name }

// TaskDiff contains the diff of two Tasks
type TaskDiff struct {
	Type        DiffType
	Name        string
	Fields      []*FieldDiff
	Objects     []*ObjectDiff
	Annotations []string
}

// Diff returns a diff of two tasks. If contextual diff is enabled, objects
// within the task will contain field information even if unchanged.
func (t *Task) Diff(other *Task, contextual bool) (*TaskDiff, error) {
	diff := &TaskDiff{Type: DiffTypeNone}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string
	filter := []string{"Name", "Config"}

	if t == nil && other == nil {
		return diff, nil
	} else if t == nil {
		t = &Task{}
		diff.Type = DiffTypeAdded
		diff.Name = other.Name
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
	} else if other == nil {
		other = &Task{}
		diff.Type = DiffTypeDeleted
		diff.Name = t.Name
		oldPrimitiveFlat = flatmap.Flatten(t, filter, true)
	} else {
		if !reflect.DeepEqual(t, other) {
			diff.Type = DiffTypeEdited
		}
		if t.Name != other.Name {
			return nil, fmt.Errorf("can not diff tasks with different names: %q and %q", t.Name, other.Name)
		}
		diff.Name = other.Name
		oldPrimitiveFlat = flatmap.Flatten(t, filter, true)
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, false)

	// Constraints diff
	conDiff := primitiveObjectSetDiff(
		interfaceSlice(t.Constraints),
		interfaceSlice(other.Constraints),
		[]string{"str"},
		"Constraint",
		contextual)
	if conDiff != nil {
		diff.Objects = append(diff.Objects, conDiff...)
	}

	// Config diff
	if cDiff := configDiff(t.Config, other.Config, contextual); cDiff != nil {
		diff.Objects = append(diff.Objects, cDiff)
	}

	// Resources diff
	if rDiff := t.Resources.Diff(other.Resources, contextual); rDiff != nil {
		diff.Objects = append(diff.Objects, rDiff)
	}

	// LogConfig diff
	lDiff := primitiveObjectDiff(t.LogConfig, other.LogConfig, nil, "LogConfig", contextual)
	if lDiff != nil {
		diff.Objects = append(diff.Objects, lDiff)
	}

	// Dispatch payload diff
	dDiff := primitiveObjectDiff(t.DispatchPayload, other.DispatchPayload, nil, "DispatchPayload", contextual)
	if dDiff != nil {
		diff.Objects = append(diff.Objects, dDiff)
	}

	// Artifacts diff
	diffs := primitiveObjectSetDiff(
		interfaceSlice(t.Artifacts),
		interfaceSlice(other.Artifacts),
		nil,
		"Artifact",
		contextual)
	if diffs != nil {
		diff.Objects = append(diff.Objects, diffs...)
	}

	// Services diff
	if sDiffs := serviceDiffs(t.Services, other.Services, contextual); sDiffs != nil {
		diff.Objects = append(diff.Objects, sDiffs...)
	}

	// Vault diff
	vDiff := vaultDiff(t.Vault, other.Vault, contextual)
	if vDiff != nil {
		diff.Objects = append(diff.Objects, vDiff)
	}

	// Artifacts diff
	tmplDiffs := primitiveObjectSetDiff(
		interfaceSlice(t.Templates),
		interfaceSlice(other.Templates),
		nil,
		"Template",
		contextual)
	if tmplDiffs != nil {
		diff.Objects = append(diff.Objects, tmplDiffs...)
	}

	return diff, nil
}

func (t *TaskDiff) GoString() string {
	var out string
	if len(t.Annotations) == 0 {
		out = fmt.Sprintf("Task %q (%s):\n", t.Name, t.Type)
	} else {
		out = fmt.Sprintf("Task %q (%s) (%s):\n", t.Name, t.Type, strings.Join(t.Annotations, ","))
	}

	for _, f := range t.Fields {
		out += fmt.Sprintf("%#v\n", f)
	}

	for _, o := range t.Objects {
		out += fmt.Sprintf("%#v\n", o)
	}

	return out
}

// taskDiffs diffs a set of tasks. If contextual diff is enabled, unchanged
// fields within objects nested in the tasks will be returned.
func taskDiffs(old, new []*Task, contextual bool) ([]*TaskDiff, error) {
	oldMap := make(map[string]*Task, len(old))
	newMap := make(map[string]*Task, len(new))
	for _, o := range old {
		oldMap[o.Name] = o
	}
	for _, n := range new {
		newMap[n.Name] = n
	}

	var diffs []*TaskDiff
	for name, oldGroup := range oldMap {
		// Diff the same, deleted and edited
		diff, err := oldGroup.Diff(newMap[name], contextual)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, diff)
	}

	for name, newGroup := range newMap {
		// Diff the added
		if old, ok := oldMap[name]; !ok {
			diff, err := old.Diff(newGroup, contextual)
			if err != nil {
				return nil, err
			}
			diffs = append(diffs, diff)
		}
	}

	sort.Sort(TaskDiffs(diffs))
	return diffs, nil
}

// For sorting TaskDiffs
type TaskDiffs []*TaskDiff

func (t TaskDiffs) Len() int           { return len(t) }
func (t TaskDiffs) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TaskDiffs) Less(i, j int) bool { return t[i].Name < t[j].Name }

// serviceDiff returns the diff of two service objects. If contextual diff is
// enabled, all fields will be returned, even if no diff occurred.
func serviceDiff(old, new *Service, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "Service"}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string

	if reflect.DeepEqual(old, new) {
		return nil
	} else if old == nil {
		old = &Service{}
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	} else if new == nil {
		new = &Service{}
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
	} else {
		diff.Type = DiffTypeEdited
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)

	// Checks diffs
	if cDiffs := serviceCheckDiffs(old.Checks, new.Checks, contextual); cDiffs != nil {
		diff.Objects = append(diff.Objects, cDiffs...)
	}

	return diff
}

// serviceDiffs diffs a set of services. If contextual diff is enabled, unchanged
// fields within objects nested in the tasks will be returned.
func serviceDiffs(old, new []*Service, contextual bool) []*ObjectDiff {
	oldMap := make(map[string]*Service, len(old))
	newMap := make(map[string]*Service, len(new))
	for _, o := range old {
		oldMap[o.Name] = o
	}
	for _, n := range new {
		newMap[n.Name] = n
	}

	var diffs []*ObjectDiff
	for name, oldService := range oldMap {
		// Diff the same, deleted and edited
		if diff := serviceDiff(oldService, newMap[name], contextual); diff != nil {
			diffs = append(diffs, diff)
		}
	}

	for name, newService := range newMap {
		// Diff the added
		if old, ok := oldMap[name]; !ok {
			if diff := serviceDiff(old, newService, contextual); diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	sort.Sort(ObjectDiffs(diffs))
	return diffs
}

// serviceCheckDiff returns the diff of two service check objects. If contextual
// diff is enabled, all fields will be returned, even if no diff occurred.
func serviceCheckDiff(old, new *ServiceCheck, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "Check"}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string

	if reflect.DeepEqual(old, new) {
		return nil
	} else if old == nil {
		old = &ServiceCheck{}
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	} else if new == nil {
		new = &ServiceCheck{}
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
	} else {
		diff.Type = DiffTypeEdited
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)
	return diff
}

// serviceCheckDiffs diffs a set of service checks. If contextual diff is
// enabled, unchanged fields within objects nested in the tasks will be
// returned.
func serviceCheckDiffs(old, new []*ServiceCheck, contextual bool) []*ObjectDiff {
	oldMap := make(map[string]*ServiceCheck, len(old))
	newMap := make(map[string]*ServiceCheck, len(new))
	for _, o := range old {
		oldMap[o.Name] = o
	}
	for _, n := range new {
		newMap[n.Name] = n
	}

	var diffs []*ObjectDiff
	for name, oldService := range oldMap {
		// Diff the same, deleted and edited
		if diff := serviceCheckDiff(oldService, newMap[name], contextual); diff != nil {
			diffs = append(diffs, diff)
		}
	}

	for name, newService := range newMap {
		// Diff the added
		if old, ok := oldMap[name]; !ok {
			if diff := serviceCheckDiff(old, newService, contextual); diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	sort.Sort(ObjectDiffs(diffs))
	return diffs
}

// vaultDiff returns the diff of two vault objects. If contextual diff is
// enabled, all fields will be returned, even if no diff occurred.
func vaultDiff(old, new *Vault, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "Vault"}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string

	if reflect.DeepEqual(old, new) {
		return nil
	} else if old == nil {
		old = &Vault{}
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	} else if new == nil {
		new = &Vault{}
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
	} else {
		diff.Type = DiffTypeEdited
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)

	// Policies diffs
	if setDiff := stringSetDiff(old.Policies, new.Policies, "Policies", contextual); setDiff != nil {
		diff.Objects = append(diff.Objects, setDiff)
	}

	return diff
}

// parameterizedJobDiff returns the diff of two parameterized job objects. If
// contextual diff is enabled, all fields will be returned, even if no diff
// occurred.
func parameterizedJobDiff(old, new *ParameterizedJobConfig, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "ParameterizedJob"}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string

	if reflect.DeepEqual(old, new) {
		return nil
	} else if old == nil {
		old = &ParameterizedJobConfig{}
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	} else if new == nil {
		new = &ParameterizedJobConfig{}
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
	} else {
		diff.Type = DiffTypeEdited
		oldPrimitiveFlat = flatmap.Flatten(old, nil, true)
		newPrimitiveFlat = flatmap.Flatten(new, nil, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)

	// Meta diffs
	if optionalDiff := stringSetDiff(old.MetaOptional, new.MetaOptional, "MetaOptional", contextual); optionalDiff != nil {
		diff.Objects = append(diff.Objects, optionalDiff)
	}

	if requiredDiff := stringSetDiff(old.MetaRequired, new.MetaRequired, "MetaRequired", contextual); requiredDiff != nil {
		diff.Objects = append(diff.Objects, requiredDiff)
	}

	return diff
}

// Diff returns a diff of two resource objects. If contextual diff is enabled,
// non-changed fields will still be returned.
func (r *Resources) Diff(other *Resources, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "Resources"}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string

	if reflect.DeepEqual(r, other) {
		return nil
	} else if r == nil {
		r = &Resources{}
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(other, nil, true)
	} else if other == nil {
		other = &Resources{}
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(r, nil, true)
	} else {
		diff.Type = DiffTypeEdited
		oldPrimitiveFlat = flatmap.Flatten(r, nil, true)
		newPrimitiveFlat = flatmap.Flatten(other, nil, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)

	// Network Resources diff
	if nDiffs := networkResourceDiffs(r.Networks, other.Networks, contextual); nDiffs != nil {
		diff.Objects = append(diff.Objects, nDiffs...)
	}

	return diff
}

// Diff returns a diff of two network resources. If contextual diff is enabled,
// non-changed fields will still be returned.
func (r *NetworkResource) Diff(other *NetworkResource, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "Network"}
	var oldPrimitiveFlat, newPrimitiveFlat map[string]string
	filter := []string{"Device", "CIDR", "IP"}

	if reflect.DeepEqual(r, other) {
		return nil
	} else if r == nil {
		r = &NetworkResource{}
		diff.Type = DiffTypeAdded
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
	} else if other == nil {
		other = &NetworkResource{}
		diff.Type = DiffTypeDeleted
		oldPrimitiveFlat = flatmap.Flatten(r, filter, true)
	} else {
		diff.Type = DiffTypeEdited
		oldPrimitiveFlat = flatmap.Flatten(r, filter, true)
		newPrimitiveFlat = flatmap.Flatten(other, filter, true)
	}

	// Diff the primitive fields.
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)

	// Port diffs
	resPorts := portDiffs(r.ReservedPorts, other.ReservedPorts, false, contextual)
	dynPorts := portDiffs(r.DynamicPorts, other.DynamicPorts, true, contextual)
	if resPorts != nil {
		diff.Objects = append(diff.Objects, resPorts...)
	}
	if dynPorts != nil {
		diff.Objects = append(diff.Objects, dynPorts...)
	}

	return diff
}

// networkResourceDiffs diffs a set of NetworkResources. If contextual diff is enabled,
// non-changed fields will still be returned.
func networkResourceDiffs(old, new []*NetworkResource, contextual bool) []*ObjectDiff {
	makeSet := func(objects []*NetworkResource) map[string]*NetworkResource {
		objMap := make(map[string]*NetworkResource, len(objects))
		for _, obj := range objects {
			hash, err := hashstructure.Hash(obj, nil)
			if err != nil {
				panic(err)
			}
			objMap[fmt.Sprintf("%d", hash)] = obj
		}

		return objMap
	}

	oldSet := makeSet(old)
	newSet := makeSet(new)

	var diffs []*ObjectDiff
	for k, oldV := range oldSet {
		if newV, ok := newSet[k]; !ok {
			if diff := oldV.Diff(newV, contextual); diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}
	for k, newV := range newSet {
		if oldV, ok := oldSet[k]; !ok {
			if diff := oldV.Diff(newV, contextual); diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	sort.Sort(ObjectDiffs(diffs))
	return diffs

}

// portDiffs returns the diff of two sets of ports. The dynamic flag marks the
// set of ports as being Dynamic ports versus Static ports. If contextual diff is enabled,
// non-changed fields will still be returned.
func portDiffs(old, new []Port, dynamic bool, contextual bool) []*ObjectDiff {
	makeSet := func(ports []Port) map[string]Port {
		portMap := make(map[string]Port, len(ports))
		for _, port := range ports {
			portMap[port.Label] = port
		}

		return portMap
	}

	oldPorts := makeSet(old)
	newPorts := makeSet(new)

	var filter []string
	name := "Static Port"
	if dynamic {
		filter = []string{"Value"}
		name = "Dynamic Port"
	}

	var diffs []*ObjectDiff
	for portLabel, oldPort := range oldPorts {
		// Diff the same, deleted and edited
		if newPort, ok := newPorts[portLabel]; ok {
			diff := primitiveObjectDiff(oldPort, newPort, filter, name, contextual)
			if diff != nil {
				diffs = append(diffs, diff)
			}
		} else {
			diff := primitiveObjectDiff(oldPort, nil, filter, name, contextual)
			if diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}
	for label, newPort := range newPorts {
		// Diff the added
		if _, ok := oldPorts[label]; !ok {
			diff := primitiveObjectDiff(nil, newPort, filter, name, contextual)
			if diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	sort.Sort(ObjectDiffs(diffs))
	return diffs

}

// configDiff returns the diff of two Task Config objects. If contextual diff is
// enabled, all fields will be returned, even if no diff occurred.
func configDiff(old, new map[string]interface{}, contextual bool) *ObjectDiff {
	diff := &ObjectDiff{Type: DiffTypeNone, Name: "Config"}
	if reflect.DeepEqual(old, new) {
		return nil
	} else if len(old) == 0 {
		diff.Type = DiffTypeAdded
	} else if len(new) == 0 {
		diff.Type = DiffTypeDeleted
	} else {
		diff.Type = DiffTypeEdited
	}

	// Diff the primitive fields.
	oldPrimitiveFlat := flatmap.Flatten(old, nil, false)
	newPrimitiveFlat := flatmap.Flatten(new, nil, false)
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)
	return diff
}

// ObjectDiff contains the diff of two generic objects.
type ObjectDiff struct {
	Type    DiffType
	Name    string
	Fields  []*FieldDiff
	Objects []*ObjectDiff
}

func (o *ObjectDiff) GoString() string {
	out := fmt.Sprintf("\n%q (%s) {\n", o.Name, o.Type)
	for _, f := range o.Fields {
		out += fmt.Sprintf("%#v\n", f)
	}
	for _, o := range o.Objects {
		out += fmt.Sprintf("%#v\n", o)
	}
	out += "}"
	return out
}

func (o *ObjectDiff) Less(other *ObjectDiff) bool {
	if reflect.DeepEqual(o, other) {
		return false
	} else if other == nil {
		return false
	} else if o == nil {
		return true
	}

	if o.Name != other.Name {
		return o.Name < other.Name
	}

	if o.Type != other.Type {
		return o.Type.Less(other.Type)
	}

	if lO, lOther := len(o.Fields), len(other.Fields); lO != lOther {
		return lO < lOther
	}

	if lO, lOther := len(o.Objects), len(other.Objects); lO != lOther {
		return lO < lOther
	}

	// Check each field
	sort.Sort(FieldDiffs(o.Fields))
	sort.Sort(FieldDiffs(other.Fields))

	for i, oV := range o.Fields {
		if oV.Less(other.Fields[i]) {
			return true
		}
	}

	// Check each object
	sort.Sort(ObjectDiffs(o.Objects))
	sort.Sort(ObjectDiffs(other.Objects))
	for i, oV := range o.Objects {
		if oV.Less(other.Objects[i]) {
			return true
		}
	}

	return false
}

// For sorting ObjectDiffs
type ObjectDiffs []*ObjectDiff

func (o ObjectDiffs) Len() int           { return len(o) }
func (o ObjectDiffs) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o ObjectDiffs) Less(i, j int) bool { return o[i].Less(o[j]) }

type FieldDiff struct {
	Type        DiffType
	Name        string
	Old, New    string
	Annotations []string
}

// fieldDiff returns a FieldDiff if old and new are different otherwise, it
// returns nil. If contextual diff is enabled, even non-changed fields will be
// returned.
func fieldDiff(old, new, name string, contextual bool) *FieldDiff {
	diff := &FieldDiff{Name: name, Type: DiffTypeNone}
	if old == new {
		if !contextual {
			return nil
		}
		diff.Old, diff.New = old, new
		return diff
	}

	if old == "" {
		diff.Type = DiffTypeAdded
		diff.New = new
	} else if new == "" {
		diff.Type = DiffTypeDeleted
		diff.Old = old
	} else {
		diff.Type = DiffTypeEdited
		diff.Old = old
		diff.New = new
	}
	return diff
}

func (f *FieldDiff) GoString() string {
	out := fmt.Sprintf("%q (%s): %q => %q", f.Name, f.Type, f.Old, f.New)
	if len(f.Annotations) != 0 {
		out += fmt.Sprintf(" (%s)", strings.Join(f.Annotations, ", "))
	}

	return out
}

func (f *FieldDiff) Less(other *FieldDiff) bool {
	if reflect.DeepEqual(f, other) {
		return false
	} else if other == nil {
		return false
	} else if f == nil {
		return true
	}

	if f.Name != other.Name {
		return f.Name < other.Name
	} else if f.Old != other.Old {
		return f.Old < other.Old
	}

	return f.New < other.New
}

// For sorting FieldDiffs
type FieldDiffs []*FieldDiff

func (f FieldDiffs) Len() int           { return len(f) }
func (f FieldDiffs) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f FieldDiffs) Less(i, j int) bool { return f[i].Less(f[j]) }

// fieldDiffs takes a map of field names to their values and returns a set of
// field diffs. If contextual diff is enabled, even non-changed fields will be
// returned.
func fieldDiffs(old, new map[string]string, contextual bool) []*FieldDiff {
	var diffs []*FieldDiff
	visited := make(map[string]struct{})
	for k, oldV := range old {
		visited[k] = struct{}{}
		newV := new[k]
		if diff := fieldDiff(oldV, newV, k, contextual); diff != nil {
			diffs = append(diffs, diff)
		}
	}

	for k, newV := range new {
		if _, ok := visited[k]; !ok {
			if diff := fieldDiff("", newV, k, contextual); diff != nil {
				diffs = append(diffs, diff)
			}
		}
	}

	sort.Sort(FieldDiffs(diffs))
	return diffs
}

// stringSetDiff diffs two sets of strings with the given name.
func stringSetDiff(old, new []string, name string, contextual bool) *ObjectDiff {
	oldMap := make(map[string]struct{}, len(old))
	newMap := make(map[string]struct{}, len(new))
	for _, o := range old {
		oldMap[o] = struct{}{}
	}
	for _, n := range new {
		newMap[n] = struct{}{}
	}
	if reflect.DeepEqual(oldMap, newMap) && !contextual {
		return nil
	}

	diff := &ObjectDiff{Name: name}
	var added, removed bool
	for k := range oldMap {
		if _, ok := newMap[k]; !ok {
			diff.Fields = append(diff.Fields, fieldDiff(k, "", name, contextual))
			removed = true
		} else if contextual {
			diff.Fields = append(diff.Fields, fieldDiff(k, k, name, contextual))
		}
	}

	for k := range newMap {
		if _, ok := oldMap[k]; !ok {
			diff.Fields = append(diff.Fields, fieldDiff("", k, name, contextual))
			added = true
		}
	}

	sort.Sort(FieldDiffs(diff.Fields))

	// Determine the type
	if added && removed {
		diff.Type = DiffTypeEdited
	} else if added {
		diff.Type = DiffTypeAdded
	} else if removed {
		diff.Type = DiffTypeDeleted
	} else {
		// Diff of an empty set
		if len(diff.Fields) == 0 {
			return nil
		}

		diff.Type = DiffTypeNone
	}

	return diff
}

// primitiveObjectDiff returns a diff of the passed objects' primitive fields.
// The filter field can be used to exclude fields from the diff. The name is the
// name of the objects. If contextual is set, non-changed fields will also be
// stored in the object diff.
func primitiveObjectDiff(old, new interface{}, filter []string, name string, contextual bool) *ObjectDiff {
	oldPrimitiveFlat := flatmap.Flatten(old, filter, true)
	newPrimitiveFlat := flatmap.Flatten(new, filter, true)
	delete(oldPrimitiveFlat, "")
	delete(newPrimitiveFlat, "")

	diff := &ObjectDiff{Name: name}
	diff.Fields = fieldDiffs(oldPrimitiveFlat, newPrimitiveFlat, contextual)

	var added, deleted, edited bool
	for _, f := range diff.Fields {
		switch f.Type {
		case DiffTypeEdited:
			edited = true
			break
		case DiffTypeDeleted:
			deleted = true
		case DiffTypeAdded:
			added = true
		}
	}

	if edited || added && deleted {
		diff.Type = DiffTypeEdited
	} else if added {
		diff.Type = DiffTypeAdded
	} else if deleted {
		diff.Type = DiffTypeDeleted
	} else {
		return nil
	}

	return diff
}

// primitiveObjectSetDiff does a set difference of the old and new sets. The
// filter parameter can be used to filter a set of primitive fields in the
// passed structs. The name corresponds to the name of the passed objects. If
// contextual diff is enabled, objects' primtive fields will be returned even if
// no diff exists.
func primitiveObjectSetDiff(old, new []interface{}, filter []string, name string, contextual bool) []*ObjectDiff {
	makeSet := func(objects []interface{}) map[string]interface{} {
		objMap := make(map[string]interface{}, len(objects))
		for _, obj := range objects {
			hash, err := hashstructure.Hash(obj, nil)
			if err != nil {
				panic(err)
			}
			objMap[fmt.Sprintf("%d", hash)] = obj
		}

		return objMap
	}

	oldSet := makeSet(old)
	newSet := makeSet(new)

	var diffs []*ObjectDiff
	for k, v := range oldSet {
		// Deleted
		if _, ok := newSet[k]; !ok {
			diffs = append(diffs, primitiveObjectDiff(v, nil, filter, name, contextual))
		}
	}
	for k, v := range newSet {
		// Added
		if _, ok := oldSet[k]; !ok {
			diffs = append(diffs, primitiveObjectDiff(nil, v, filter, name, contextual))
		}
	}

	sort.Sort(ObjectDiffs(diffs))
	return diffs
}

// interfaceSlice is a helper method that takes a slice of typed elements and
// returns a slice of interface. This method will panic if given a non-slice
// input.
func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}
