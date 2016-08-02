package terraform

import "sync"

// MockResourceProvider implements ResourceProvider but mocks out all the
// calls for testing purposes.
type MockResourceProvider struct {
	sync.Mutex

	// Anything you want, in case you need to store extra data with the mock.
	Meta interface{}

	CloseCalled                    bool
	CloseError                     error
	InputCalled                    bool
	InputInput                     UIInput
	InputConfig                    *ResourceConfig
	InputReturnConfig              *ResourceConfig
	InputReturnError               error
	InputFn                        func(UIInput, *ResourceConfig) (*ResourceConfig, error)
	ApplyCalled                    bool
	ApplyInfo                      *InstanceInfo
	ApplyState                     *InstanceState
	ApplyDiff                      *InstanceDiff
	ApplyFn                        func(*InstanceInfo, *InstanceState, *InstanceDiff) (*InstanceState, error)
	ApplyReturn                    *InstanceState
	ApplyReturnError               error
	ConfigureCalled                bool
	ConfigureConfig                *ResourceConfig
	ConfigureFn                    func(*ResourceConfig) error
	ConfigureReturnError           error
	DiffCalled                     bool
	DiffInfo                       *InstanceInfo
	DiffState                      *InstanceState
	DiffDesired                    *ResourceConfig
	DiffFn                         func(*InstanceInfo, *InstanceState, *ResourceConfig) (*InstanceDiff, error)
	DiffReturn                     *InstanceDiff
	DiffReturnError                error
	RefreshCalled                  bool
	RefreshInfo                    *InstanceInfo
	RefreshState                   *InstanceState
	RefreshFn                      func(*InstanceInfo, *InstanceState) (*InstanceState, error)
	RefreshReturn                  *InstanceState
	RefreshReturnError             error
	ResourcesCalled                bool
	ResourcesReturn                []ResourceType
	ReadDataApplyCalled            bool
	ReadDataApplyInfo              *InstanceInfo
	ReadDataApplyDiff              *InstanceDiff
	ReadDataApplyFn                func(*InstanceInfo, *InstanceDiff) (*InstanceState, error)
	ReadDataApplyReturn            *InstanceState
	ReadDataApplyReturnError       error
	ReadDataDiffCalled             bool
	ReadDataDiffInfo               *InstanceInfo
	ReadDataDiffDesired            *ResourceConfig
	ReadDataDiffFn                 func(*InstanceInfo, *ResourceConfig) (*InstanceDiff, error)
	ReadDataDiffReturn             *InstanceDiff
	ReadDataDiffReturnError        error
	DataSourcesCalled              bool
	DataSourcesReturn              []DataSource
	ValidateCalled                 bool
	ValidateConfig                 *ResourceConfig
	ValidateFn                     func(*ResourceConfig) ([]string, []error)
	ValidateReturnWarns            []string
	ValidateReturnErrors           []error
	ValidateResourceFn             func(string, *ResourceConfig) ([]string, []error)
	ValidateResourceCalled         bool
	ValidateResourceType           string
	ValidateResourceConfig         *ResourceConfig
	ValidateResourceReturnWarns    []string
	ValidateResourceReturnErrors   []error
	ValidateDataSourceFn           func(string, *ResourceConfig) ([]string, []error)
	ValidateDataSourceCalled       bool
	ValidateDataSourceType         string
	ValidateDataSourceConfig       *ResourceConfig
	ValidateDataSourceReturnWarns  []string
	ValidateDataSourceReturnErrors []error

	ImportStateCalled      bool
	ImportStateInfo        *InstanceInfo
	ImportStateID          string
	ImportStateReturn      []*InstanceState
	ImportStateReturnError error
	ImportStateFn          func(*InstanceInfo, string) ([]*InstanceState, error)
}

func (p *MockResourceProvider) Close() error {
	p.CloseCalled = true
	return p.CloseError
}

func (p *MockResourceProvider) Input(
	input UIInput, c *ResourceConfig) (*ResourceConfig, error) {
	p.InputCalled = true
	p.InputInput = input
	p.InputConfig = c
	if p.InputFn != nil {
		return p.InputFn(input, c)
	}
	return p.InputReturnConfig, p.InputReturnError
}

func (p *MockResourceProvider) Validate(c *ResourceConfig) ([]string, []error) {
	p.Lock()
	defer p.Unlock()

	p.ValidateCalled = true
	p.ValidateConfig = c
	if p.ValidateFn != nil {
		return p.ValidateFn(c)
	}
	return p.ValidateReturnWarns, p.ValidateReturnErrors
}

func (p *MockResourceProvider) ValidateResource(t string, c *ResourceConfig) ([]string, []error) {
	p.Lock()
	defer p.Unlock()

	p.ValidateResourceCalled = true
	p.ValidateResourceType = t
	p.ValidateResourceConfig = c

	if p.ValidateResourceFn != nil {
		return p.ValidateResourceFn(t, c)
	}

	return p.ValidateResourceReturnWarns, p.ValidateResourceReturnErrors
}

func (p *MockResourceProvider) Configure(c *ResourceConfig) error {
	p.Lock()
	defer p.Unlock()

	p.ConfigureCalled = true
	p.ConfigureConfig = c

	if p.ConfigureFn != nil {
		return p.ConfigureFn(c)
	}

	return p.ConfigureReturnError
}

func (p *MockResourceProvider) Apply(
	info *InstanceInfo,
	state *InstanceState,
	diff *InstanceDiff) (*InstanceState, error) {
	// We only lock while writing data. Reading is fine
	p.Lock()
	p.ApplyCalled = true
	p.ApplyInfo = info
	p.ApplyState = state
	p.ApplyDiff = diff
	p.Unlock()

	if p.ApplyFn != nil {
		return p.ApplyFn(info, state, diff)
	}

	return p.ApplyReturn, p.ApplyReturnError
}

func (p *MockResourceProvider) Diff(
	info *InstanceInfo,
	state *InstanceState,
	desired *ResourceConfig) (*InstanceDiff, error) {
	p.Lock()
	defer p.Unlock()

	p.DiffCalled = true
	p.DiffInfo = info
	p.DiffState = state
	p.DiffDesired = desired
	if p.DiffFn != nil {
		return p.DiffFn(info, state, desired)
	}

	return p.DiffReturn, p.DiffReturnError
}

func (p *MockResourceProvider) Refresh(
	info *InstanceInfo,
	s *InstanceState) (*InstanceState, error) {
	p.Lock()
	defer p.Unlock()

	p.RefreshCalled = true
	p.RefreshInfo = info
	p.RefreshState = s

	if p.RefreshFn != nil {
		return p.RefreshFn(info, s)
	}

	return p.RefreshReturn, p.RefreshReturnError
}

func (p *MockResourceProvider) Resources() []ResourceType {
	p.Lock()
	defer p.Unlock()

	p.ResourcesCalled = true
	return p.ResourcesReturn
}

func (p *MockResourceProvider) ImportState(info *InstanceInfo, id string) ([]*InstanceState, error) {
	p.Lock()
	defer p.Unlock()

	p.ImportStateCalled = true
	p.ImportStateInfo = info
	p.ImportStateID = id
	if p.ImportStateFn != nil {
		return p.ImportStateFn(info, id)
	}

	return p.ImportStateReturn, p.ImportStateReturnError
}

func (p *MockResourceProvider) ValidateDataSource(t string, c *ResourceConfig) ([]string, []error) {
	p.Lock()
	defer p.Unlock()

	p.ValidateDataSourceCalled = true
	p.ValidateDataSourceType = t
	p.ValidateDataSourceConfig = c

	if p.ValidateDataSourceFn != nil {
		return p.ValidateDataSourceFn(t, c)
	}

	return p.ValidateDataSourceReturnWarns, p.ValidateDataSourceReturnErrors
}

func (p *MockResourceProvider) ReadDataDiff(
	info *InstanceInfo,
	desired *ResourceConfig) (*InstanceDiff, error) {
	p.Lock()
	defer p.Unlock()

	p.ReadDataDiffCalled = true
	p.ReadDataDiffInfo = info
	p.ReadDataDiffDesired = desired
	if p.ReadDataDiffFn != nil {
		return p.ReadDataDiffFn(info, desired)
	}

	return p.ReadDataDiffReturn, p.ReadDataDiffReturnError
}

func (p *MockResourceProvider) ReadDataApply(
	info *InstanceInfo,
	d *InstanceDiff) (*InstanceState, error) {
	p.Lock()
	defer p.Unlock()

	p.ReadDataApplyCalled = true
	p.ReadDataApplyInfo = info
	p.ReadDataApplyDiff = d

	if p.ReadDataApplyFn != nil {
		return p.ReadDataApplyFn(info, d)
	}

	return p.ReadDataApplyReturn, p.ReadDataApplyReturnError
}

func (p *MockResourceProvider) DataSources() []DataSource {
	p.Lock()
	defer p.Unlock()

	p.DataSourcesCalled = true
	return p.DataSourcesReturn
}
