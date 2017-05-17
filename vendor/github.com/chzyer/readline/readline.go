// Readline is a pure go implementation for GNU-Readline kind library.
//
// example:
// 	rl, err := readline.New("> ")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer rl.Close()
//
// 	for {
// 		line, err := rl.Readline()
// 		if err != nil { // io.EOF
// 			break
// 		}
// 		println(line)
// 	}
//
package readline

import "io"

type Instance struct {
	Config    *Config
	Terminal  *Terminal
	Operation *Operation
}

type Config struct {
	// prompt supports ANSI escape sequence, so we can color some characters even in windows
	Prompt string

	// readline will persist historys to file where HistoryFile specified
	HistoryFile string
	// specify the max length of historys, it's 500 by default, set it to -1 to disable history
	HistoryLimit           int
	DisableAutoSaveHistory bool
	// enable case-insensitive history searching
	HistorySearchFold bool

	// AutoCompleter will called once user press TAB
	AutoComplete AutoCompleter

	// Any key press will pass to Listener
	// NOTE: Listener will be triggered by (nil, 0, 0) immediately
	Listener Listener

	// If VimMode is true, readline will in vim.insert mode by default
	VimMode bool

	InterruptPrompt string
	EOFPrompt       string

	FuncGetWidth func() int

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	EnableMask bool
	MaskRune   rune

	// erase the editing line after user submited it
	// it use in IM usually.
	UniqueEditLine bool

	// force use interactive even stdout is not a tty
	FuncIsTerminal      func() bool
	FuncMakeRaw         func() error
	FuncExitRaw         func() error
	FuncOnWidthChanged  func(func())
	ForceUseInteractive bool

	// private fields
	inited    bool
	opHistory *opHistory
	opSearch  *opSearch
}

func (c *Config) useInteractive() bool {
	if c.ForceUseInteractive {
		return true
	}
	return c.FuncIsTerminal()
}

func (c *Config) Init() error {
	if c.inited {
		return nil
	}
	c.inited = true
	if c.Stdin == nil {
		c.Stdin = NewCancelableStdin(Stdin)
	}
	if c.Stdout == nil {
		c.Stdout = Stdout
	}
	if c.Stderr == nil {
		c.Stderr = Stderr
	}
	if c.HistoryLimit == 0 {
		c.HistoryLimit = 500
	}

	if c.InterruptPrompt == "" {
		c.InterruptPrompt = "^C"
	} else if c.InterruptPrompt == "\n" {
		c.InterruptPrompt = ""
	}
	if c.EOFPrompt == "" {
		c.EOFPrompt = "^D"
	} else if c.EOFPrompt == "\n" {
		c.EOFPrompt = ""
	}

	if c.AutoComplete == nil {
		c.AutoComplete = &TabCompleter{}
	}
	if c.FuncGetWidth == nil {
		c.FuncGetWidth = GetScreenWidth
	}
	if c.FuncIsTerminal == nil {
		c.FuncIsTerminal = DefaultIsTerminal
	}
	rm := new(RawMode)
	if c.FuncMakeRaw == nil {
		c.FuncMakeRaw = rm.Enter
	}
	if c.FuncExitRaw == nil {
		c.FuncExitRaw = rm.Exit
	}
	if c.FuncOnWidthChanged == nil {
		c.FuncOnWidthChanged = DefaultOnWidthChanged
	}

	return nil
}

func (c Config) Clone() *Config {
	c.opHistory = nil
	c.opSearch = nil
	return &c
}

func (c *Config) SetListener(f func(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool)) {
	c.Listener = FuncListener(f)
}

func NewEx(cfg *Config) (*Instance, error) {
	t, err := NewTerminal(cfg)
	if err != nil {
		return nil, err
	}
	rl := t.Readline()
	return &Instance{
		Config:    cfg,
		Terminal:  t,
		Operation: rl,
	}, nil
}

func New(prompt string) (*Instance, error) {
	return NewEx(&Config{Prompt: prompt})
}

func (i *Instance) ResetHistory() {
	i.Operation.ResetHistory()
}

func (i *Instance) SetPrompt(s string) {
	i.Operation.SetPrompt(s)
}

func (i *Instance) SetMaskRune(r rune) {
	i.Operation.SetMaskRune(r)
}

// change history persistence in runtime
func (i *Instance) SetHistoryPath(p string) {
	i.Operation.SetHistoryPath(p)
}

// readline will refresh automatic when write through Stdout()
func (i *Instance) Stdout() io.Writer {
	return i.Operation.Stdout()
}

// readline will refresh automatic when write through Stdout()
func (i *Instance) Stderr() io.Writer {
	return i.Operation.Stderr()
}

// switch VimMode in runtime
func (i *Instance) SetVimMode(on bool) {
	i.Operation.SetVimMode(on)
}

func (i *Instance) IsVimMode() bool {
	return i.Operation.IsEnableVimMode()
}

func (i *Instance) GenPasswordConfig() *Config {
	return i.Operation.GenPasswordConfig()
}

// we can generate a config by `i.GenPasswordConfig()`
func (i *Instance) ReadPasswordWithConfig(cfg *Config) ([]byte, error) {
	return i.Operation.PasswordWithConfig(cfg)
}

func (i *Instance) ReadPasswordEx(prompt string, l Listener) ([]byte, error) {
	return i.Operation.PasswordEx(prompt, l)
}

func (i *Instance) ReadPassword(prompt string) ([]byte, error) {
	return i.Operation.Password(prompt)
}

type Result struct {
	Line  string
	Error error
}

func (l *Result) CanContinue() bool {
	return len(l.Line) != 0 && l.Error == ErrInterrupt
}

func (l *Result) CanBreak() bool {
	return !l.CanContinue() && l.Error != nil
}

func (i *Instance) Line() *Result {
	ret, err := i.Readline()
	return &Result{ret, err}
}

// err is one of (nil, io.EOF, readline.ErrInterrupt)
func (i *Instance) Readline() (string, error) {
	return i.Operation.String()
}

func (i *Instance) SaveHistory(content string) error {
	return i.Operation.SaveHistory(content)
}

// same as readline
func (i *Instance) ReadSlice() ([]byte, error) {
	return i.Operation.Slice()
}

// we must make sure that call Close() before process exit.
func (i *Instance) Close() error {
	if err := i.Terminal.Close(); err != nil {
		return err
	}
	i.Operation.Close()
	return nil
}
func (i *Instance) Clean() {
	i.Operation.Clean()
}

func (i *Instance) Write(b []byte) (int, error) {
	return i.Stdout().Write(b)
}

func (i *Instance) SetConfig(cfg *Config) *Config {
	if i.Config == cfg {
		return cfg
	}
	old := i.Config
	i.Config = cfg
	i.Operation.SetConfig(cfg)
	i.Terminal.SetConfig(cfg)
	return old
}

func (i *Instance) Refresh() {
	i.Operation.Refresh()
}
