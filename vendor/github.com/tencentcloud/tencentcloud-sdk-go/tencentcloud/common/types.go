package common

func IntPtr(v int) *int {
	return &v
}

func Int64Ptr(v int64) *int64 {
	return &v
}

func UintPtr(v uint) *uint {
	return &v
}

func Uint64Ptr(v uint64) *uint64 {
	return &v
}

func Float64Ptr(v float64) *float64 {
	return &v
}

func StringPtr(v string) *string {
	return &v
}

func StringValues(ptrs []*string) []string {
	values := make([]string, len(ptrs))
	for i := 0; i < len(ptrs); i++ {
		if ptrs[i] != nil {
			values[i] = *ptrs[i]
		}
	}
	return values
}

func StringPtrs(vals []string) []*string {
	ptrs := make([]*string, len(vals))
	for i := 0; i < len(vals); i++ {
		ptrs[i] = &vals[i]
	}
	return ptrs
}

func BoolPtr(v bool) *bool {
	return &v
}
