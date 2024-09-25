# Removed Block

- Config Declaration: internal/stacks/stackconfig/removed.go (L39)
- Config Parsing: internal/stacks/stackconfig/file.go (L231 & L140)
- Config Loading: internal/s tacks/stackconfig/config.go (L121 & L195)
- Static Walk: internal/stacks/stackruntime/internal/stackeval/walk_static.go (L61)
    - Visit (for plan): internal/stacks/stackruntime/internal/stackeval/main_plan.go (L108)
    - PlanChanges: internal/stacks/stackruntime/internal/stackeval/removed_config.go (L213)
- Dynamic Walk: internal/stacks/stackruntime/internal/stackeval/walk_dynamic.go (L184)
    - removed.PlanChanges: internal/stacks/stackruntime/internal/stackeval/removed.go (L206)
    - removed.Instances: internal/stacks/stackruntime/internal/stackeval/removed.go (L100)
    - removedInstance.PlanChanges: internal/stacks/stackruntime/internal/stackeval/removed_instance.go (L246)
