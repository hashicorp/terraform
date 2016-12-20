package arukas

import (
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
	"runtime"
)

// ExitCode is exit code.
var ExitCode = 0

var (
	cli = kingpin.New("arukas", "A CLI for Arukas Cloud")

	ps        = cli.Command("ps", "Show status of containers")
	psListAll = ps.Flag("all", "Show all containers (default shows just running)").Short('a').Bool()
	psQuiet   = ps.Flag("quiet", "Only display numeric IDs").Short('q').Bool()

	rm            = cli.Command("rm", "Remove a container")
	rmContainerID = rm.Arg("container_id", "Container ID").Required().String()

	run          = cli.Command("run", "Create and run a container. The container must run as a daemon.")
	runImage     = run.Arg("image", "Image").Required().String()
	runInstances = run.Flag("instances", "Number of instances").Required().Int()
	runMem       = run.Flag("mem", "Memory size").Required().Int()
	runAppName   = run.Flag("app-name", "The name of the app.").String()
	runName      = run.Flag("name", "The name of container which must be unique").String()
	runCmd       = run.Flag("cmd", "Command to execute").String()
	runEnvs      = run.Flag("envs", "Set environment variables. -e KEY=VALUE").Short('e').Strings()
	runPorts     = run.Flag("ports", "Publish a container's port(s) to the internet. -p 80:tcp").Short('p').Required().Strings()

	start            = cli.Command("start", "Start one stopped container")
	startContainerID = start.Arg("container_id", "Container ID").Required().String()

	stop            = cli.Command("stop", "Stop one running container")
	stopContainerID = stop.Arg("container_id", "Container ID").Required().String()

	version = cli.Command("version", "Print version information and quit")
)

// Run arukas
func Run(args []string) int {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(VERSION)
	switch kingpin.MustParse(cli.Parse(args[1:])) {
	case "ps":
		listContainers(*psListAll, *psQuiet)
	case "rm":
		removeContainer(*rmContainerID)
	case "run":
		createAndRunContainer(*runName, *runImage, *runInstances, *runMem, *runEnvs, *runPorts, *runCmd, *runAppName)
	case "start":
		startContainer(*startContainerID, false)
	case "stop":
		stopContainer(*stopContainerID)
	case "version":
		displayVersion()
	}

	return ExitCode
}

// RunTest arukas
func RunTest(args []string) int {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logFile := "/tmp/test.log"
	if runtime.GOOS != "windows" {
		// logFile = ".test.log"
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err != nil {
			log.Fatal("error opening file :", err.Error())
		}

		log.SetOutput(f)
	}
	return Run(args)
}
