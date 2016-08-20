package state

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/nomad/nomad/structs"
)

// stateStoreSchema is used to return the schema for the state store
func stateStoreSchema() *memdb.DBSchema {
	// Create the root DB schema
	db := &memdb.DBSchema{
		Tables: make(map[string]*memdb.TableSchema),
	}

	// Collect all the schemas that are needed
	schemas := []func() *memdb.TableSchema{
		indexTableSchema,
		nodeTableSchema,
		jobTableSchema,
		jobSummarySchema,
		periodicLaunchTableSchema,
		evalTableSchema,
		allocTableSchema,
	}

	// Add each of the tables
	for _, schemaFn := range schemas {
		schema := schemaFn()
		if _, ok := db.Tables[schema.Name]; ok {
			panic(fmt.Sprintf("duplicate table name: %s", schema.Name))
		}
		db.Tables[schema.Name] = schema
	}
	return db
}

// indexTableSchema is used for
func indexTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "index",
		Indexes: map[string]*memdb.IndexSchema{
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field:     "Key",
					Lowercase: true,
				},
			},
		},
	}
}

// nodeTableSchema returns the MemDB schema for the nodes table.
// This table is used to store all the client nodes that are registered.
func nodeTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "nodes",
		Indexes: map[string]*memdb.IndexSchema{
			// Primary index is used for node management
			// and simple direct lookup. ID is required to be
			// unique.
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.UUIDFieldIndex{
					Field: "ID",
				},
			},
		},
	}
}

// jobTableSchema returns the MemDB schema for the jobs table.
// This table is used to store all the jobs that have been submitted.
func jobTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "jobs",
		Indexes: map[string]*memdb.IndexSchema{
			// Primary index is used for job management
			// and simple direct lookup. ID is required to be
			// unique.
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field:     "ID",
					Lowercase: true,
				},
			},
			"type": &memdb.IndexSchema{
				Name:         "type",
				AllowMissing: false,
				Unique:       false,
				Indexer: &memdb.StringFieldIndex{
					Field:     "Type",
					Lowercase: false,
				},
			},
			"gc": &memdb.IndexSchema{
				Name:         "gc",
				AllowMissing: false,
				Unique:       false,
				Indexer: &memdb.ConditionalIndex{
					Conditional: jobIsGCable,
				},
			},
			"periodic": &memdb.IndexSchema{
				Name:         "periodic",
				AllowMissing: false,
				Unique:       false,
				Indexer: &memdb.ConditionalIndex{
					Conditional: jobIsPeriodic,
				},
			},
		},
	}
}

// jobSummarySchema returns the memdb schema for the job summary table
func jobSummarySchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "job_summary",
		Indexes: map[string]*memdb.IndexSchema{
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field:     "JobID",
					Lowercase: true,
				},
			},
		},
	}
}

// jobIsGCable satisfies the ConditionalIndexFunc interface and creates an index
// on whether a job is eligible for garbage collection.
func jobIsGCable(obj interface{}) (bool, error) {
	j, ok := obj.(*structs.Job)
	if !ok {
		return false, fmt.Errorf("Unexpected type: %v", obj)
	}

	// The job is GCable if it is batch and it is not periodic
	periodic := j.Periodic != nil && j.Periodic.Enabled
	gcable := j.Type == structs.JobTypeBatch && !periodic
	return gcable, nil
}

// jobIsPeriodic satisfies the ConditionalIndexFunc interface and creates an index
// on whether a job is periodic.
func jobIsPeriodic(obj interface{}) (bool, error) {
	j, ok := obj.(*structs.Job)
	if !ok {
		return false, fmt.Errorf("Unexpected type: %v", obj)
	}

	if j.Periodic != nil && j.Periodic.Enabled == true {
		return true, nil
	}

	return false, nil
}

// periodicLaunchTableSchema returns the MemDB schema tracking the most recent
// launch time for a perioidic job.
func periodicLaunchTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "periodic_launch",
		Indexes: map[string]*memdb.IndexSchema{
			// Primary index is used for job management
			// and simple direct lookup. ID is required to be
			// unique.
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field:     "ID",
					Lowercase: true,
				},
			},
		},
	}
}

// evalTableSchema returns the MemDB schema for the eval table.
// This table is used to store all the evaluations that are pending
// or recently completed.
func evalTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "evals",
		Indexes: map[string]*memdb.IndexSchema{
			// Primary index is used for direct lookup.
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.UUIDFieldIndex{
					Field: "ID",
				},
			},

			// Job index is used to lookup allocations by job
			"job": &memdb.IndexSchema{
				Name:         "job",
				AllowMissing: false,
				Unique:       false,
				Indexer: &memdb.StringFieldIndex{
					Field:     "JobID",
					Lowercase: true,
				},
			},
		},
	}
}

// allocTableSchema returns the MemDB schema for the allocation table.
// This table is used to store all the task allocations between task groups
// and nodes.
func allocTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "allocs",
		Indexes: map[string]*memdb.IndexSchema{
			// Primary index is a UUID
			"id": &memdb.IndexSchema{
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.UUIDFieldIndex{
					Field: "ID",
				},
			},

			// Node index is used to lookup allocations by node
			"node": &memdb.IndexSchema{
				Name:         "node",
				AllowMissing: true, // Missing is allow for failed allocations
				Unique:       false,
				Indexer: &memdb.CompoundIndex{
					Indexes: []memdb.Indexer{
						&memdb.StringFieldIndex{
							Field:     "NodeID",
							Lowercase: true,
						},

						// Conditional indexer on if allocation is terminal
						&memdb.ConditionalIndex{
							Conditional: func(obj interface{}) (bool, error) {
								// Cast to allocation
								alloc, ok := obj.(*structs.Allocation)
								if !ok {
									return false, fmt.Errorf("wrong type, got %t should be Allocation", obj)
								}

								// Check if the allocation is terminal
								return alloc.TerminalStatus(), nil
							},
						},
					},
				},
			},

			// Job index is used to lookup allocations by job
			"job": &memdb.IndexSchema{
				Name:         "job",
				AllowMissing: false,
				Unique:       false,
				Indexer: &memdb.StringFieldIndex{
					Field:     "JobID",
					Lowercase: true,
				},
			},

			// Eval index is used to lookup allocations by eval
			"eval": &memdb.IndexSchema{
				Name:         "eval",
				AllowMissing: false,
				Unique:       false,
				Indexer: &memdb.UUIDFieldIndex{
					Field: "EvalID",
				},
			},
		},
	}
}
