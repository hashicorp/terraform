===============================================================================
GopherLua: VM and compiler for Lua in Go.
===============================================================================

.. image:: https://godoc.org/github.com/yuin/gopher-lua?status.svg
    :target: http://godoc.org/github.com/yuin/gopher-lua

.. image:: https://travis-ci.org/yuin/gopher-lua.svg
    :target: https://travis-ci.org/yuin/gopher-lua

.. image:: https://coveralls.io/repos/yuin/gopher-lua/badge.svg
    :target: https://coveralls.io/r/yuin/gopher-lua

.. image:: https://badges.gitter.im/Join%20Chat.svg
    :alt: Join the chat at https://gitter.im/yuin/gopher-lua
    :target: https://gitter.im/yuin/gopher-lua?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge

|


GopherLua is a Lua5.1 VM and compiler written in Go. GopherLua has a same goal
with Lua: **Be a scripting language with extensible semantics** . It provides
Go APIs that allow you to easily embed a scripting language to your Go host
programs.

.. contents::
   :depth: 1

----------------------------------------------------------------
Design principle
----------------------------------------------------------------

- Be a scripting language with extensible semantics.
- User-friendly Go API
    - The stack based API like the one used in the original Lua
      implementation will cause a performance improvements in GopherLua
      (It will reduce memory allocations and concrete type <-> interface conversions).
      GopherLua API is **not** the stack based API.
      GopherLua give preference to the user-friendliness over the performance.

----------------------------------------------------------------
How about performance?
----------------------------------------------------------------
GopherLua is not fast but not too slow, I think.

GopherLua has almost equivalent ( or little bit better ) performance as Python3 on micro benchmarks.

There are some benchmarks on the `wiki page <https://github.com/yuin/gopher-lua/wiki/Benchmarks>`_ .

----------------------------------------------------------------
Installation
----------------------------------------------------------------

.. code-block:: bash

   go get github.com/yuin/gopher-lua

GopherLua supports >= Go1.7.

----------------------------------------------------------------
Usage
----------------------------------------------------------------
GopherLua APIs perform in much the same way as Lua, **but the stack is used only
for passing arguments and receiving returned values.**

GopherLua supports channel operations. See **"Goroutines"** section.

Import a package.

.. code-block:: go

   import (
       "github.com/yuin/gopher-lua"
   )

Run scripts in the VM.

.. code-block:: go

   L := lua.NewState()
   defer L.Close()
   if err := L.DoString(`print("hello")`); err != nil {
       panic(err)
   }

.. code-block:: go

   L := lua.NewState()
   defer L.Close()
   if err := L.DoFile("hello.lua"); err != nil {
       panic(err)
   }

Refer to `Lua Reference Manual <http://www.lua.org/manual/5.1/>`_ and `Go doc <http://godoc.org/github.com/yuin/gopher-lua>`_ for further information.

Note that elements that are not commented in `Go doc <http://godoc.org/github.com/yuin/gopher-lua>`_ equivalent to `Lua Reference Manual <http://www.lua.org/manual/5.1/>`_ , except GopherLua uses objects instead of Lua stack indices.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Data model
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
All data in a GopherLua program is an ``LValue`` . ``LValue`` is an interface
type that has following methods.

- ``String() string``
- ``Type() LValueType``


Objects implement an LValue interface are

================ ========================= ================== =======================
 Type name        Go type                   Type() value       Constants
================ ========================= ================== =======================
 ``LNilType``      (constants)              ``LTNil``          ``LNil``
 ``LBool``         (constants)              ``LTBool``         ``LTrue``, ``LFalse``
 ``LNumber``        float64                 ``LTNumber``       ``-``
 ``LString``        string                  ``LTString``       ``-``
 ``LFunction``      struct pointer          ``LTFunction``     ``-``
 ``LUserData``      struct pointer          ``LTUserData``     ``-``
 ``LState``         struct pointer          ``LTThread``       ``-``
 ``LTable``         struct pointer          ``LTTable``        ``-``
 ``LChannel``       chan LValue             ``LTChannel``      ``-``
================ ========================= ================== =======================

You can test an object type in Go way(type assertion) or using a ``Type()`` value.

.. code-block:: go

   lv := L.Get(-1) // get the value at the top of the stack
   if str, ok := lv.(lua.LString); ok {
       // lv is LString
       fmt.Println(string(str))
   }
   if lv.Type() != lua.LTString {
       panic("string required.")
   }

.. code-block:: go

   lv := L.Get(-1) // get the value at the top of the stack
   if tbl, ok := lv.(*lua.LTable); ok {
       // lv is LTable
       fmt.Println(L.ObjLen(tbl))
   }

Note that ``LBool`` , ``LNumber`` , ``LString`` is not a pointer.

To test ``LNilType`` and ``LBool``, You **must** use pre-defined constants.

.. code-block:: go

   lv := L.Get(-1) // get the value at the top of the stack

   if lv == lua.LTrue { // correct
   }

   if bl, ok := lv.(lua.LBool); ok && bool(bl) { // wrong
   }

In Lua, both ``nil`` and ``false`` make a condition false. ``LVIsFalse`` and ``LVAsBool`` implement this specification.

.. code-block:: go

   lv := L.Get(-1) // get the value at the top of the stack
   if lua.LVIsFalse(lv) { // lv is nil or false
   }

   if lua.LVAsBool(lv) { // lv is neither nil nor false
   }

Objects that based on go structs(``LFunction``. ``LUserData``, ``LTable``)
have some public methods and fields. You can use these methods and fields for
performance and debugging, but there are some limitations.

- Metatable does not work.
- No error handlings.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Callstack & Registry size
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Size of the callstack & registry is **fixed** for mainly performance.
You can change the default size of the callstack & registry.

.. code-block:: go

   lua.RegistrySize = 1024 * 20
   lua.CallStackSize = 1024
   L := lua.NewState()
   defer L.Close()

You can also create an LState object that has the callstack & registry size specified by ``Options`` .

.. code-block:: go

    L := lua.NewState(lua.Options{
        CallStackSize: 120,
        RegistrySize:  120*20,
    })

An LState object that has been created by ``*LState#NewThread()`` inherits the callstack & registry size from the parent LState object.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Miscellaneous lua.NewState options
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
- **Options.SkipOpenLibs bool(default false)**
    - By default, GopherLua opens all built-in libraries when new LState is created.
    - You can skip this behaviour by setting this to ``true`` .
    - Using the various `OpenXXX(L *LState) int` functions you can open only those libraries that you require, for an example see below.
- **Options.IncludeGoStackTrace bool(default false)**
    - By default, GopherLua does not show Go stack traces when panics occur.
    - You can get Go stack traces by setting this to ``true`` .

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
API
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Refer to `Lua Reference Manual <http://www.lua.org/manual/5.1/>`_ and `Go doc(LState methods) <http://godoc.org/github.com/yuin/gopher-lua>`_ for further information.

+++++++++++++++++++++++++++++++++++++++++
Calling Go from Lua
+++++++++++++++++++++++++++++++++++++++++

.. code-block:: go

   func Double(L *lua.LState) int {
       lv := L.ToInt(1)             /* get argument */
       L.Push(lua.LNumber(lv * 2)) /* push result */
       return 1                     /* number of results */
   }

   func main() {
       L := lua.NewState()
       defer L.Close()
       L.SetGlobal("double", L.NewFunction(Double)) /* Original lua_setglobal uses stack... */
   }

.. code-block:: lua

   print(double(20)) -- > "40"

Any function registered with GopherLua is a ``lua.LGFunction``, defined in ``value.go``

.. code-block:: go

   type LGFunction func(*LState) int

Working with coroutines.

.. code-block:: go

   co, _ := L.NewThread() /* create a new thread */
   fn := L.GetGlobal("coro").(*lua.LFunction) /* get function from lua */
   for {
       st, err, values := L.Resume(co, fn)
       if st == lua.ResumeError {
           fmt.Println("yield break(error)")
           fmt.Println(err.Error())
           break
       }

       for i, lv := range values {
           fmt.Printf("%v : %v\n", i, lv)
       }

       if st == lua.ResumeOK {
           fmt.Println("yield break(ok)")
           break
       }
   }

+++++++++++++++++++++++++++++++++++++++++
Opening a subset of builtin modules
+++++++++++++++++++++++++++++++++++++++++

The following demonstrates how to open a subset of the built-in modules in Lua, say for example to avoid enabling modules with access to local files or system calls.

main.go

.. code-block:: go

    func main() {
        L := lua.NewState(lua.Options{SkipOpenLibs: true})
        defer L.Close()
        for _, pair := range []struct {
            n string
            f lua.LGFunction
        }{
            {lua.LoadLibName, lua.OpenPackage}, // Must be first
            {lua.BaseLibName, lua.OpenBase},
            {lua.TabLibName, lua.OpenTable},
        } {
            if err := L.CallByParam(lua.P{
                Fn:      L.NewFunction(pair.f),
                NRet:    0,
                Protect: true,
            }, lua.LString(pair.n)); err != nil {
                panic(err)
            }
        }
        if err := L.DoFile("main.lua"); err != nil {
            panic(err)
        }
    }

+++++++++++++++++++++++++++++++++++++++++
Creating a module by Go
+++++++++++++++++++++++++++++++++++++++++

mymodule.go

.. code-block:: go

    package mymodule

    import (
        "github.com/yuin/gopher-lua"
    )

    func Loader(L *lua.LState) int {
        // register functions to the table
        mod := L.SetFuncs(L.NewTable(), exports)
        // register other stuff
        L.SetField(mod, "name", lua.LString("value"))

        // returns the module
        L.Push(mod)
        return 1
    }

    var exports = map[string]lua.LGFunction{
        "myfunc": myfunc,
    }

    func myfunc(L *lua.LState) int {
        return 0
    }

mymain.go

.. code-block:: go

    package main

    import (
        "./mymodule"
        "github.com/yuin/gopher-lua"
    )

    func main() {
        L := lua.NewState()
        defer L.Close()
        L.PreloadModule("mymodule", mymodule.Loader)
        if err := L.DoFile("main.lua"); err != nil {
            panic(err)
        }
    }

main.lua

.. code-block:: lua

    local m = require("mymodule")
    m.myfunc()
    print(m.name)


+++++++++++++++++++++++++++++++++++++++++
Calling Lua from Go
+++++++++++++++++++++++++++++++++++++++++

.. code-block:: go

   L := lua.NewState()
   defer L.Close()
   if err := L.DoFile("double.lua"); err != nil {
       panic(err)
   }
   if err := L.CallByParam(lua.P{
       Fn: L.GetGlobal("double"),
       NRet: 1,
       Protect: true,
       }, lua.LNumber(10)); err != nil {
       panic(err)
   }
   ret := L.Get(-1) // returned value
   L.Pop(1)  // remove received value

If ``Protect`` is false, GopherLua will panic instead of returning an ``error`` value.

+++++++++++++++++++++++++++++++++++++++++
User-Defined types
+++++++++++++++++++++++++++++++++++++++++
You can extend GopherLua with new types written in Go.
``LUserData`` is provided for this purpose.

.. code-block:: go

    type Person struct {
        Name string
    }

    const luaPersonTypeName = "person"

    // Registers my person type to given L.
    func registerPersonType(L *lua.LState) {
        mt := L.NewTypeMetatable(luaPersonTypeName)
        L.SetGlobal("person", mt)
        // static attributes
        L.SetField(mt, "new", L.NewFunction(newPerson))
        // methods
        L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), personMethods))
    }

    // Constructor
    func newPerson(L *lua.LState) int {
        person := &Person{L.CheckString(1)}
        ud := L.NewUserData()
        ud.Value = person
        L.SetMetatable(ud, L.GetTypeMetatable(luaPersonTypeName))
        L.Push(ud)
        return 1
    }

    // Checks whether the first lua argument is a *LUserData with *Person and returns this *Person.
    func checkPerson(L *lua.LState) *Person {
        ud := L.CheckUserData(1)
        if v, ok := ud.Value.(*Person); ok {
            return v
        }
        L.ArgError(1, "person expected")
        return nil
    }

    var personMethods = map[string]lua.LGFunction{
        "name": personGetSetName,
    }

    // Getter and setter for the Person#Name
    func personGetSetName(L *lua.LState) int {
        p := checkPerson(L)
        if L.GetTop() == 2 {
            p.Name = L.CheckString(2)
            return 0
        }
        L.Push(lua.LString(p.Name))
        return 1
    }

    func main() {
        L := lua.NewState()
        defer L.Close()
        registerPersonType(L)
        if err := L.DoString(`
            p = person.new("Steeve")
            print(p:name()) -- "Steeve"
            p:name("Alice")
            print(p:name()) -- "Alice"
        `); err != nil {
            panic(err)
        }
    }

+++++++++++++++++++++++++++++++++++++++++
Terminating a running LState
+++++++++++++++++++++++++++++++++++++++++
GopherLua supports the `Go Concurrency Patterns: Context <https://blog.golang.org/context>`_ .


.. code-block:: go

    L := lua.NewState()
    defer L.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    // set the context to our LState
    L.SetContext(ctx)
    err := L.DoString(`
      local clock = os.clock
      function sleep(n)  -- seconds
        local t0 = clock()
        while clock() - t0 <= n do end
      end
      sleep(3)
    `)
    // err.Error() contains "context deadline exceeded"

With coroutines

.. code-block:: go

	L := lua.NewState()
	defer L.Close()
	ctx, cancel := context.WithCancel(context.Background())
	L.SetContext(ctx)
	defer cancel()
	L.DoString(`
	    function coro()
		  local i = 0
		  while true do
		    coroutine.yield(i)
			i = i+1
		  end
		  return i
	    end
	`)
	co, cocancel := L.NewThread()
	defer cocancel()
	fn := L.GetGlobal("coro").(*LFunction)
    
	_, err, values := L.Resume(co, fn) // err is nil
    
	cancel() // cancel the parent context
    
	_, err, values = L.Resume(co, fn) // err is NOT nil : child context was canceled

**Note that using a context causes performance degradation.**

.. code-block::

    time ./glua-with-context.exe fib.lua
    9227465
    0.01s user 0.11s system 1% cpu 7.505 total

    time ./glua-without-context.exe fib.lua
    9227465
    0.01s user 0.01s system 0% cpu 5.306 total


+++++++++++++++++++++++++++++++++++++++++
Goroutines
+++++++++++++++++++++++++++++++++++++++++
The ``LState`` is not goroutine-safe. It is recommended to use one LState per goroutine and communicate between goroutines by using channels.

Channels are represented by ``channel`` objects in GopherLua. And a ``channel`` table provides functions for performing channel operations.

Some objects can not be sent over channels due to having non-goroutine-safe objects inside itself.

- a thread(state)
- a function
- an userdata
- a table with a metatable

You **must not** send these objects from Go APIs to channels.



.. code-block:: go

    func receiver(ch, quit chan lua.LValue) {
        L := lua.NewState()
        defer L.Close()
        L.SetGlobal("ch", lua.LChannel(ch))
        L.SetGlobal("quit", lua.LChannel(quit))
        if err := L.DoString(`
        local exit = false
        while not exit do
          channel.select(
            {"|<-", ch, function(ok, v)
              if not ok then
                print("channel closed")
                exit = true
              else
                print("received:", v)
              end
            end},
            {"|<-", quit, function(ok, v)
                print("quit")
                exit = true
            end}
          )
        end
      `); err != nil {
            panic(err)
        }
    }

    func sender(ch, quit chan lua.LValue) {
        L := lua.NewState()
        defer L.Close()
        L.SetGlobal("ch", lua.LChannel(ch))
        L.SetGlobal("quit", lua.LChannel(quit))
        if err := L.DoString(`
        ch:send("1")
        ch:send("2")
      `); err != nil {
            panic(err)
        }
        ch <- lua.LString("3")
        quit <- lua.LTrue
    }

    func main() {
        ch := make(chan lua.LValue)
        quit := make(chan lua.LValue)
        go receiver(ch, quit)
        go sender(ch, quit)
        time.Sleep(3 * time.Second)
    }

'''''''''''''''
Go API
'''''''''''''''

``ToChannel``, ``CheckChannel``, ``OptChannel`` are available.

Refer to `Go doc(LState methods) <http://godoc.org/github.com/yuin/gopher-lua>`_ for further information.

'''''''''''''''
Lua API
'''''''''''''''

- **channel.make([buf:int]) -> ch:channel**
    - Create new channel that has a buffer size of ``buf``. By default, ``buf`` is 0.

- **channel.select(case:table [, case:table, case:table ...]) -> {index:int, recv:any, ok}**
    - Same as the ``select`` statement in Go. It returns the index of the chosen case and, if that
      case was a receive operation, the value received and a boolean indicating whether the channel has been closed.
    - ``case`` is a table that outlined below.
        - receiving: `{"|<-", ch:channel [, handler:func(ok, data:any)]}`
        - sending: `{"<-|", ch:channel, data:any [, handler:func(data:any)]}`
        - default: `{"default" [, handler:func()]}`

``channel.select`` examples:

.. code-block:: lua

    local idx, recv, ok = channel.select(
      {"|<-", ch1},
      {"|<-", ch2}
    )
    if not ok then
        print("closed")
    elseif idx == 1 then -- received from ch1
        print(recv)
    elseif idx == 2 then -- received from ch2
        print(recv)
    end

.. code-block:: lua

    channel.select(
      {"|<-", ch1, function(ok, data)
        print(ok, data)
      end},
      {"<-|", ch2, "value", function(data)
        print(data)
      end},
      {"default", function()
        print("default action")
      end}
    )

- **channel:send(data:any)**
    - Send ``data`` over the channel.
- **channel:receive() -> ok:bool, data:any**
    - Receive some data over the channel.
- **channel:close()**
    - Close the channel.

''''''''''''''''''''''''''''''
The LState pool pattern
''''''''''''''''''''''''''''''
To create per-thread LState instances, You can use the ``sync.Pool`` like mechanism.

.. code-block:: go

    type lStatePool struct {
        m     sync.Mutex
        saved []*lua.LState
    }

    func (pl *lStatePool) Get() *lua.LState {
        pl.m.Lock()
        defer pl.m.Unlock()
        n := len(pl.saved)
        if n == 0 {
            return pl.New()
        }
        x := pl.saved[n-1]
        pl.saved = pl.saved[0 : n-1]
        return x
    }

    func (pl *lStatePool) New() *lua.LState {
        L := lua.NewState()
        // setting the L up here.
        // load scripts, set global variables, share channels, etc...
        return L
    }

    func (pl *lStatePool) Put(L *lua.LState) {
        pl.m.Lock()
        defer pl.m.Unlock()
        pl.saved = append(pl.saved, L)
    }

    func (pl *lStatePool) Shutdown() {
        for _, L := range pl.saved {
            L.Close()
        }
    }

    // Global LState pool
    var luaPool = &lStatePool{
        saved: make([]*lua.LState, 0, 4),
    }

Now, you can get per-thread LState objects from the ``luaPool`` .

.. code-block:: go

    func MyWorker() {
       L := luaPool.Get()
       defer luaPool.Put(L)
       /* your code here */
    }

    func main() {
        defer luaPool.Shutdown()
        go MyWorker()
        go MyWorker()
        /* etc... */
    }


----------------------------------------------------------------
Differences between Lua and GopherLua
----------------------------------------------------------------
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Goroutines
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

- GopherLua supports channel operations.
    - GopherLua has a type named ``channel``.
    - The ``channel`` table provides functions for performing channel operations.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Unsupported functions
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

- ``string.dump``
- ``os.setlocale``
- ``lua_Debug.namewhat``
- ``package.loadlib``
- debug hooks

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Miscellaneous notes
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

- ``collectgarbage`` does not take any arguments and runs the garbage collector for the entire Go program.
- ``file:setvbuf`` does not support a line buffering.
- Daylight saving time is not supported.
- GopherLua has a function to set an environment variable : ``os.setenv(name, value)``

----------------------------------------------------------------
Standalone interpreter
----------------------------------------------------------------
Lua has an interpreter called ``lua`` . GopherLua has an interpreter called ``glua`` .

.. code-block:: bash

   go get github.com/yuin/gopher-lua/cmd/glua

``glua`` has same options as ``lua`` .

----------------------------------------------------------------
How to Contribute
----------------------------------------------------------------
See `Guidlines for contributors <https://github.com/yuin/gopher-lua/tree/master/.github/CONTRIBUTING.md>`_ .

----------------------------------------------------------------
Libraries for GopherLua
----------------------------------------------------------------

- `gopher-luar <https://github.com/layeh/gopher-luar>`_ : Custom type reflection for gopher-lua
- `gluamapper <https://github.com/yuin/gluamapper>`_ : Mapping a Lua table to a Go struct
- `gluare <https://github.com/yuin/gluare>`_ : Regular expressions for gopher-lua
- `gluahttp <https://github.com/cjoudrey/gluahttp>`_ : HTTP request module for gopher-lua
- `gopher-json <https://github.com/layeh/gopher-json>`_ : A simple JSON encoder/decoder for gopher-lua
- `gluayaml <https://github.com/kohkimakimoto/gluayaml>`_ : Yaml parser for gopher-lua
- `glua-lfs <https://github.com/layeh/gopher-lfs>`_ : Partially implements the luafilesystem module for gopher-lua
- `gluaurl <https://github.com/cjoudrey/gluaurl>`_ : A url parser/builder module for gopher-lua
- `gluahttpscrape <https://github.com/felipejfc/gluahttpscrape>`_ : A simple HTML scraper module for gopher-lua
- `gluaxmlpath <https://github.com/ailncode/gluaxmlpath>`_ : An xmlpath module for gopher-lua
- `gluasocket <https://github.com/BixData/gluasocket>`_ : A LuaSocket library for the GopherLua VM

----------------------------------------------------------------
Donation
----------------------------------------------------------------

BTC: 1NEDSyUmo4SMTDP83JJQSWi1MvQUGGNMZB

----------------------------------------------------------------
License
----------------------------------------------------------------
MIT

----------------------------------------------------------------
Author
----------------------------------------------------------------
Yusuke Inuzuka
