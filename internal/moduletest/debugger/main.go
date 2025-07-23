package debugger

// func main() {
// 	listener, err := net.Listen("tcp", ":5368")
// 	if err != nil {
// 		panic(err)
// 	}

// 	defer listener.Close()
// 	fmt.Printf("Go server listening on Unix domain socket: %s\n", listener.Addr())

// 	src := "/Users/sams/hashicorp/terraform/modules-dev/parallel/main.tftest.hcl"
// 	// modsDir := filepath.Join(path.Base(src), ".terraform/modules")
// 	// testingDir := filepath.Join(path.Base(src), "tests")
// 	// loader, err := configload.NewLoader(&configload.Config{ModulesDir: modsDir})
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// config, hclDiags := loader.LoadConfigWithTests("/Users/sams/hashicorp/terraform/modules-dev/parallel", "tests")
// 	// if hclDiags.HasErrors() {
// 	// 	panic(hclDiags.Error())
// 	// }

// 	// runningCtx, done := context.WithCancel(context.Background())
// 	// stopCtx, stop := context.WithCancel(runningCtx)
// 	// cancelCtx, cancel := context.WithCancel(context.Background())

// 	// for _, run := range config.Module.Tests[path.Base(src)].Runs {
// 	// 	runs[run.DeclRange.Start.Line] = run.DeclRange.Start
// 	// }

// 	// localRunner := &local.TestSuiteRunner{
// 	// 	Config: config,
// 	// 	// The GlobalVariables are loaded from the
// 	// 	// main configuration directory
// 	// 	// The GlobalTestVariables are loaded from the
// 	// 	// test directory
// 	// 	// GlobalVariables:     variables,
// 	// 	// GlobalTestVariables: testVariables,
// 	// 	TestingDirectory: testingDir,
// 	// 	Opts:             opts,
// 	// 	View:             &views.TestJSON{},
// 	// 	Stopped:          false,
// 	// 	Cancelled:        false,
// 	// 	StoppedCtx:       stopCtx,
// 	// 	CancelledCtx:     cancelCtx,
// 	// 	// Filter:           args.Filter,
// 	// 	// Verbose:          args.Verbose,
// 	// 	Concurrency: 1,
// 	// }

// 	ctx := context.Background()
// 	err = NewServer(&DebugSession{
// 		Source: src,
// 		Step:   1,
// 		State:  make(map[string]any),
// 	}).Serve(ctx, listener)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// func NewDebugSession() *DebugSession {
// 	listener, err := net.Listen("tcp", ":5368")
// 	if err != nil {
// 		panic(err)
// 	}

// 	defer listener.Close()
// 	fmt.Printf("Go server listening on Unix domain socket: %s\n", listener.Addr())

// 	src := "/Users/sams/hashicorp/terraform/modules-dev/parallel/main.tftest.hcl"
// 	ctx := context.Background()
// 	err = NewServer(&DebugSession{
// 		Source: src,
// 		Step:   1,
// 		State:  make(map[string]any),
// 	}).Serve(ctx, listener)
// 	if err != nil {
// 		panic(err)
// 	}
// }
