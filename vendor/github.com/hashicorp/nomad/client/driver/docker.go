package driver

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/nomad/client/allocdir"
	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/client/driver/executor"
	dstructs "github.com/hashicorp/nomad/client/driver/structs"
	cstructs "github.com/hashicorp/nomad/client/structs"
	"github.com/hashicorp/nomad/helper/discover"
	"github.com/hashicorp/nomad/helper/fields"
	shelpers "github.com/hashicorp/nomad/helper/stats"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/mapstructure"
)

var (
	// We store the clients globally to cache the connection to the docker daemon.
	createClients sync.Once

	// client is a docker client with a timeout of 1 minute. This is for doing
	// all operations with the docker daemon besides which are not long running
	// such as creating, killing containers, etc.
	client *docker.Client

	// waitClient is a docker client with no timeouts. This is used for long
	// running operations such as waiting on containers and collect stats
	waitClient *docker.Client

	// The statistics the Docker driver exposes
	DockerMeasuredMemStats = []string{"RSS", "Cache", "Swap", "Max Usage"}
	DockerMeasuredCpuStats = []string{"Throttled Periods", "Throttled Time", "Percent"}
)

const (
	// NoSuchContainerError is returned by the docker daemon if the container
	// does not exist.
	NoSuchContainerError = "No such container"

	// The key populated in Node Attributes to indicate presence of the Docker
	// driver
	dockerDriverAttr = "driver.docker"

	// dockerTimeout is the length of time a request can be outstanding before
	// it is timed out.
	dockerTimeout = 1 * time.Minute
)

type DockerDriver struct {
	DriverContext
}

type DockerDriverAuth struct {
	Username      string `mapstructure:"username"`       // username for the registry
	Password      string `mapstructure:"password"`       // password to access the registry
	Email         string `mapstructure:"email"`          // email address of the user who is allowed to access the registry
	ServerAddress string `mapstructure:"server_address"` // server address of the registry
}

type DockerDriverConfig struct {
	ImageName        string              `mapstructure:"image"`              // Container's Image Name
	LoadImages       []string            `mapstructure:"load"`               // LoadImage is array of paths to image archive files
	Command          string              `mapstructure:"command"`            // The Command/Entrypoint to run when the container starts up
	Args             []string            `mapstructure:"args"`               // The arguments to the Command/Entrypoint
	IpcMode          string              `mapstructure:"ipc_mode"`           // The IPC mode of the container - host and none
	NetworkMode      string              `mapstructure:"network_mode"`       // The network mode of the container - host, nat and none
	PidMode          string              `mapstructure:"pid_mode"`           // The PID mode of the container - host and none
	UTSMode          string              `mapstructure:"uts_mode"`           // The UTS mode of the container - host and none
	PortMapRaw       []map[string]int    `mapstructure:"port_map"`           //
	PortMap          map[string]int      `mapstructure:"-"`                  // A map of host port labels and the ports exposed on the container
	Privileged       bool                `mapstructure:"privileged"`         // Flag to run the container in privileged mode
	DNSServers       []string            `mapstructure:"dns_servers"`        // DNS Server for containers
	DNSSearchDomains []string            `mapstructure:"dns_search_domains"` // DNS Search domains for containers
	Hostname         string              `mapstructure:"hostname"`           // Hostname for containers
	LabelsRaw        []map[string]string `mapstructure:"labels"`             //
	Labels           map[string]string   `mapstructure:"-"`                  // Labels to set when the container starts up
	Auth             []DockerDriverAuth  `mapstructure:"auth"`               // Authentication credentials for a private Docker registry
	SSL              bool                `mapstructure:"ssl"`                // Flag indicating repository is served via https
	TTY              bool                `mapstructure:"tty"`                // Allocate a Pseudo-TTY
	Interactive      bool                `mapstructure:"interactive"`        // Keep STDIN open even if not attached
	ShmSize          int64               `mapstructure:"shm_size"`           // Size of /dev/shm of the container in bytes
	WorkDir          string              `mapstructure:"work_dir"`           // Working directory inside the container
}

// Validate validates a docker driver config
func (c *DockerDriverConfig) Validate() error {
	if c.ImageName == "" {
		return fmt.Errorf("Docker Driver needs an image name")
	}

	c.PortMap = mapMergeStrInt(c.PortMapRaw...)
	c.Labels = mapMergeStrStr(c.LabelsRaw...)

	return nil
}

// NewDockerDriverConfig returns a docker driver config by parsing the HCL
// config
func NewDockerDriverConfig(task *structs.Task) (*DockerDriverConfig, error) {
	var driverConfig DockerDriverConfig
	driverConfig.SSL = true
	if err := mapstructure.WeakDecode(task.Config, &driverConfig); err != nil {
		return nil, err
	}
	if strings.Contains(driverConfig.ImageName, "https://") {
		driverConfig.ImageName = strings.Replace(driverConfig.ImageName, "https://", "", 1)
	}

	if err := driverConfig.Validate(); err != nil {
		return nil, err
	}
	return &driverConfig, nil
}

type dockerPID struct {
	Version        string
	ImageID        string
	ContainerID    string
	KillTimeout    time.Duration
	MaxKillTimeout time.Duration
	PluginConfig   *PluginReattachConfig
}

type DockerHandle struct {
	pluginClient      *plugin.Client
	executor          executor.Executor
	client            *docker.Client
	waitClient        *docker.Client
	logger            *log.Logger
	cleanupImage      bool
	imageID           string
	containerID       string
	version           string
	clkSpeed          float64
	killTimeout       time.Duration
	maxKillTimeout    time.Duration
	resourceUsageLock sync.RWMutex
	resourceUsage     *cstructs.TaskResourceUsage
	waitCh            chan *dstructs.WaitResult
	doneCh            chan bool
}

func NewDockerDriver(ctx *DriverContext) Driver {
	return &DockerDriver{DriverContext: *ctx}
}

// Validate is used to validate the driver configuration
func (d *DockerDriver) Validate(config map[string]interface{}) error {
	fd := &fields.FieldData{
		Raw: config,
		Schema: map[string]*fields.FieldSchema{
			"image": &fields.FieldSchema{
				Type:     fields.TypeString,
				Required: true,
			},
			"load": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"command": &fields.FieldSchema{
				Type: fields.TypeString,
			},
			"args": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"ipc_mode": &fields.FieldSchema{
				Type: fields.TypeString,
			},
			"network_mode": &fields.FieldSchema{
				Type: fields.TypeString,
			},
			"pid_mode": &fields.FieldSchema{
				Type: fields.TypeString,
			},
			"uts_mode": &fields.FieldSchema{
				Type: fields.TypeString,
			},
			"port_map": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"privileged": &fields.FieldSchema{
				Type: fields.TypeBool,
			},
			"dns_servers": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"dns_search_domains": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"hostname": &fields.FieldSchema{
				Type: fields.TypeString,
			},
			"labels": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"auth": &fields.FieldSchema{
				Type: fields.TypeArray,
			},
			"ssl": &fields.FieldSchema{
				Type: fields.TypeBool,
			},
			"tty": &fields.FieldSchema{
				Type: fields.TypeBool,
			},
			"interactive": &fields.FieldSchema{
				Type: fields.TypeBool,
			},
			"shm_size": &fields.FieldSchema{
				Type: fields.TypeInt,
			},
			"work_dir": &fields.FieldSchema{
				Type: fields.TypeString,
			},
		},
	}

	if err := fd.Validate(); err != nil {
		return err
	}

	return nil
}

// dockerClients creates two *docker.Client, one for long running operations and
// the other for shorter operations. In test / dev mode we can use ENV vars to
// connect to the docker daemon. In production mode we will read docker.endpoint
// from the config file.
func (d *DockerDriver) dockerClients() (*docker.Client, *docker.Client, error) {
	if client != nil && waitClient != nil {
		return client, waitClient, nil
	}

	var err error
	var merr multierror.Error
	createClients.Do(func() {
		if err = shelpers.Init(); err != nil {
			d.logger.Printf("[FATAL] driver.docker: unable to initialize stats: %v", err)
			return
		}

		// Default to using whatever is configured in docker.endpoint. If this is
		// not specified we'll fall back on NewClientFromEnv which reads config from
		// the DOCKER_* environment variables DOCKER_HOST, DOCKER_TLS_VERIFY, and
		// DOCKER_CERT_PATH. This allows us to lock down the config in production
		// but also accept the standard ENV configs for dev and test.
		dockerEndpoint := d.config.Read("docker.endpoint")
		if dockerEndpoint != "" {
			cert := d.config.Read("docker.tls.cert")
			key := d.config.Read("docker.tls.key")
			ca := d.config.Read("docker.tls.ca")

			if cert+key+ca != "" {
				d.logger.Printf("[DEBUG] driver.docker: using TLS client connection to %s", dockerEndpoint)
				client, err = docker.NewTLSClient(dockerEndpoint, cert, key, ca)
				if err != nil {
					merr.Errors = append(merr.Errors, err)
				}
				waitClient, err = docker.NewTLSClient(dockerEndpoint, cert, key, ca)
				if err != nil {
					merr.Errors = append(merr.Errors, err)
				}
			} else {
				d.logger.Printf("[DEBUG] driver.docker: using standard client connection to %s", dockerEndpoint)
				client, err = docker.NewClient(dockerEndpoint)
				if err != nil {
					merr.Errors = append(merr.Errors, err)
				}
				waitClient, err = docker.NewClient(dockerEndpoint)
				if err != nil {
					merr.Errors = append(merr.Errors, err)
				}
			}
			client.SetTimeout(dockerTimeout)
			return
		}

		d.logger.Println("[DEBUG] driver.docker: using client connection initialized from environment")
		client, err = docker.NewClientFromEnv()
		if err != nil {
			merr.Errors = append(merr.Errors, err)
		}
		client.SetTimeout(dockerTimeout)

		waitClient, err = docker.NewClientFromEnv()
		if err != nil {
			merr.Errors = append(merr.Errors, err)
		}
	})
	return client, waitClient, merr.ErrorOrNil()
}

func (d *DockerDriver) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	// Get the current status so that we can log any debug messages only if the
	// state changes
	_, currentlyEnabled := node.Attributes[dockerDriverAttr]

	// Initialize docker API clients
	client, _, err := d.dockerClients()
	if err != nil {
		delete(node.Attributes, dockerDriverAttr)
		if currentlyEnabled {
			d.logger.Printf("[INFO] driver.docker: failed to initialize client: %s", err)
		}
		return false, nil
	}

	privileged := d.config.ReadBoolDefault("docker.privileged.enabled", false)
	if privileged {
		node.Attributes["docker.privileged.enabled"] = "1"
	}

	// This is the first operation taken on the client so we'll try to
	// establish a connection to the Docker daemon. If this fails it means
	// Docker isn't available so we'll simply disable the docker driver.
	env, err := client.Version()
	if err != nil {
		if currentlyEnabled {
			d.logger.Printf("[DEBUG] driver.docker: could not connect to docker daemon at %s: %s", client.Endpoint(), err)
		}
		delete(node.Attributes, dockerDriverAttr)
		return false, nil
	}

	node.Attributes[dockerDriverAttr] = "1"
	node.Attributes["driver.docker.version"] = env.Get("Version")
	return true, nil
}

func (d *DockerDriver) containerBinds(alloc *allocdir.AllocDir, task *structs.Task) ([]string, error) {
	shared := alloc.SharedDir
	local, ok := alloc.TaskDirs[task.Name]
	if !ok {
		return nil, fmt.Errorf("Failed to find task local directory: %v", task.Name)
	}

	allocDirBind := fmt.Sprintf("%s:%s", shared, allocdir.SharedAllocContainerPath)
	taskLocalBind := fmt.Sprintf("%s:%s", local, allocdir.TaskLocalContainerPath)

	if selinuxLabel := d.config.Read("docker.volumes.selinuxlabel"); selinuxLabel != "" {
		allocDirBind = fmt.Sprintf("%s:%s", allocDirBind, selinuxLabel)
		taskLocalBind = fmt.Sprintf("%s:%s", taskLocalBind, selinuxLabel)
	}
	return []string{
		allocDirBind,
		taskLocalBind,
	}, nil
}

// createContainer initializes a struct needed to call docker.client.CreateContainer()
func (d *DockerDriver) createContainer(ctx *ExecContext, task *structs.Task,
	driverConfig *DockerDriverConfig, syslogAddr string) (docker.CreateContainerOptions, error) {
	var c docker.CreateContainerOptions
	if task.Resources == nil {
		// Guard against missing resources. We should never have been able to
		// schedule a job without specifying this.
		d.logger.Println("[ERR] driver.docker: task.Resources is empty")
		return c, fmt.Errorf("task.Resources is empty")
	}

	binds, err := d.containerBinds(ctx.AllocDir, task)
	if err != nil {
		return c, err
	}

	// Set environment variables.
	d.taskEnv.SetAllocDir(allocdir.SharedAllocContainerPath)
	d.taskEnv.SetTaskLocalDir(allocdir.TaskLocalContainerPath)

	config := &docker.Config{
		Image:     driverConfig.ImageName,
		Hostname:  driverConfig.Hostname,
		User:      task.User,
		Tty:       driverConfig.TTY,
		OpenStdin: driverConfig.Interactive,
	}

	if driverConfig.WorkDir != "" {
		config.WorkingDir = driverConfig.WorkDir
	}

	memLimit := int64(task.Resources.MemoryMB) * 1024 * 1024
	hostConfig := &docker.HostConfig{
		// Convert MB to bytes. This is an absolute value.
		Memory:     memLimit,
		MemorySwap: memLimit, // MemorySwap is memory + swap.
		// Convert Mhz to shares. This is a relative value.
		CPUShares: int64(task.Resources.CPU),

		// Binds are used to mount a host volume into the container. We mount a
		// local directory for storage and a shared alloc directory that can be
		// used to share data between different tasks in the same task group.
		Binds: binds,
		LogConfig: docker.LogConfig{
			Type: "syslog",
			Config: map[string]string{
				"syslog-address": syslogAddr,
			},
		},
	}

	d.logger.Printf("[DEBUG] driver.docker: using %d bytes memory for %s", hostConfig.Memory, task.Config["image"])
	d.logger.Printf("[DEBUG] driver.docker: using %d cpu shares for %s", hostConfig.CPUShares, task.Config["image"])
	d.logger.Printf("[DEBUG] driver.docker: binding directories %#v for %s", hostConfig.Binds, task.Config["image"])

	//  set privileged mode
	hostPrivileged := d.config.ReadBoolDefault("docker.privileged.enabled", false)
	if driverConfig.Privileged && !hostPrivileged {
		return c, fmt.Errorf(`Docker privileged mode is disabled on this Nomad agent`)
	}
	hostConfig.Privileged = driverConfig.Privileged

	// set SHM size
	if driverConfig.ShmSize != 0 {
		hostConfig.ShmSize = driverConfig.ShmSize
	}

	// set DNS servers
	for _, ip := range driverConfig.DNSServers {
		if net.ParseIP(ip) != nil {
			hostConfig.DNS = append(hostConfig.DNS, ip)
		} else {
			d.logger.Printf("[ERR] driver.docker: invalid ip address for container dns server: %s", ip)
		}
	}

	// set DNS search domains
	for _, domain := range driverConfig.DNSSearchDomains {
		hostConfig.DNSSearch = append(hostConfig.DNSSearch, domain)
	}

	hostConfig.IpcMode = driverConfig.IpcMode
	hostConfig.PidMode = driverConfig.PidMode
	hostConfig.UTSMode = driverConfig.UTSMode

	hostConfig.NetworkMode = driverConfig.NetworkMode
	if hostConfig.NetworkMode == "" {
		// docker default
		d.logger.Printf("[DEBUG] driver.docker: networking mode not specified; defaulting to %s", defaultNetworkMode)
		hostConfig.NetworkMode = defaultNetworkMode
	}

	// Setup port mapping and exposed ports
	if len(task.Resources.Networks) == 0 {
		d.logger.Println("[DEBUG] driver.docker: No network interfaces are available")
		if len(driverConfig.PortMap) > 0 {
			return c, fmt.Errorf("Trying to map ports but no network interface is available")
		}
	} else {
		// TODO add support for more than one network
		network := task.Resources.Networks[0]
		publishedPorts := map[docker.Port][]docker.PortBinding{}
		exposedPorts := map[docker.Port]struct{}{}

		for _, port := range network.ReservedPorts {
			// By default we will map the allocated port 1:1 to the container
			containerPortInt := port.Value

			// If the user has mapped a port using port_map we'll change it here
			if mapped, ok := driverConfig.PortMap[port.Label]; ok {
				containerPortInt = mapped
			}

			hostPortStr := strconv.Itoa(port.Value)
			containerPort := docker.Port(strconv.Itoa(containerPortInt))

			publishedPorts[containerPort+"/tcp"] = getPortBinding(network.IP, hostPortStr)
			publishedPorts[containerPort+"/udp"] = getPortBinding(network.IP, hostPortStr)
			d.logger.Printf("[DEBUG] driver.docker: allocated port %s:%d -> %d (static)", network.IP, port.Value, port.Value)

			exposedPorts[containerPort+"/tcp"] = struct{}{}
			exposedPorts[containerPort+"/udp"] = struct{}{}
			d.logger.Printf("[DEBUG] driver.docker: exposed port %d", port.Value)
		}

		for _, port := range network.DynamicPorts {
			// By default we will map the allocated port 1:1 to the container
			containerPortInt := port.Value

			// If the user has mapped a port using port_map we'll change it here
			if mapped, ok := driverConfig.PortMap[port.Label]; ok {
				containerPortInt = mapped
			}

			hostPortStr := strconv.Itoa(port.Value)
			containerPort := docker.Port(strconv.Itoa(containerPortInt))

			publishedPorts[containerPort+"/tcp"] = getPortBinding(network.IP, hostPortStr)
			publishedPorts[containerPort+"/udp"] = getPortBinding(network.IP, hostPortStr)
			d.logger.Printf("[DEBUG] driver.docker: allocated port %s:%d -> %d (mapped)", network.IP, port.Value, containerPortInt)

			exposedPorts[containerPort+"/tcp"] = struct{}{}
			exposedPorts[containerPort+"/udp"] = struct{}{}
			d.logger.Printf("[DEBUG] driver.docker: exposed port %s", containerPort)
		}

		d.taskEnv.SetPortMap(driverConfig.PortMap)

		hostConfig.PortBindings = publishedPorts
		config.ExposedPorts = exposedPorts
	}

	d.taskEnv.Build()
	parsedArgs := d.taskEnv.ParseAndReplace(driverConfig.Args)

	// If the user specified a custom command to run as their entrypoint, we'll
	// inject it here.
	if driverConfig.Command != "" {
		// Validate command
		if err := validateCommand(driverConfig.Command, "args"); err != nil {
			return c, err
		}

		cmd := []string{driverConfig.Command}
		if len(driverConfig.Args) != 0 {
			cmd = append(cmd, parsedArgs...)
		}
		d.logger.Printf("[DEBUG] driver.docker: setting container startup command to: %s", strings.Join(cmd, " "))
		config.Cmd = cmd
	} else if len(driverConfig.Args) != 0 {
		config.Cmd = parsedArgs
	}

	if len(driverConfig.Labels) > 0 {
		config.Labels = driverConfig.Labels
		d.logger.Printf("[DEBUG] driver.docker: applied labels on the container: %+v", config.Labels)
	}

	config.Env = d.taskEnv.EnvList()

	containerName := fmt.Sprintf("%s-%s", task.Name, ctx.AllocID)
	d.logger.Printf("[DEBUG] driver.docker: setting container name to: %s", containerName)

	return docker.CreateContainerOptions{
		Name:       containerName,
		Config:     config,
		HostConfig: hostConfig,
	}, nil
}

var (
	// imageNotFoundMatcher is a regex expression that matches the image not
	// found error Docker returns.
	imageNotFoundMatcher = regexp.MustCompile(`Error: image .+ not found`)
)

// recoverablePullError wraps the error gotten when trying to pull and image if
// the error is recoverable.
func (d *DockerDriver) recoverablePullError(err error, image string) error {
	recoverable := true
	if imageNotFoundMatcher.MatchString(err.Error()) {
		recoverable = false
	}
	return dstructs.NewRecoverableError(fmt.Errorf("Failed to pull `%s`: %s", image, err), recoverable)
}

func (d *DockerDriver) Periodic() (bool, time.Duration) {
	return true, 15 * time.Second
}

// createImage creates a docker image either by pulling it from a registry or by
// loading it from the file system
func (d *DockerDriver) createImage(driverConfig *DockerDriverConfig, client *docker.Client, taskDir string) error {
	image := driverConfig.ImageName
	repo, tag := docker.ParseRepositoryTag(image)
	if tag == "" {
		tag = "latest"
	}

	var dockerImage *docker.Image
	var err error
	// We're going to check whether the image is already downloaded. If the tag
	// is "latest" we have to check for a new version every time so we don't
	// bother to check and cache the id here. We'll download first, then cache.
	if tag != "latest" {
		dockerImage, err = client.InspectImage(image)
	}

	// Download the image
	if dockerImage == nil {
		if len(driverConfig.LoadImages) > 0 {
			return d.loadImage(driverConfig, client, taskDir)
		}

		return d.pullImage(driverConfig, client, repo, tag)
	}
	return err
}

// pullImage creates an image by pulling it from a docker registry
func (d *DockerDriver) pullImage(driverConfig *DockerDriverConfig, client *docker.Client, repo string, tag string) error {
	pullOptions := docker.PullImageOptions{
		Repository: repo,
		Tag:        tag,
	}

	authOptions := docker.AuthConfiguration{}
	if len(driverConfig.Auth) != 0 {
		authOptions = docker.AuthConfiguration{
			Username:      driverConfig.Auth[0].Username,
			Password:      driverConfig.Auth[0].Password,
			Email:         driverConfig.Auth[0].Email,
			ServerAddress: driverConfig.Auth[0].ServerAddress,
		}
	}

	if authConfigFile := d.config.Read("docker.auth.config"); authConfigFile != "" {
		if f, err := os.Open(authConfigFile); err == nil {
			defer f.Close()
			var authConfigurations *docker.AuthConfigurations
			if authConfigurations, err = docker.NewAuthConfigurations(f); err != nil {
				return fmt.Errorf("Failed to create docker auth object: %v", err)
			}

			authConfigurationKey := ""
			if driverConfig.SSL {
				authConfigurationKey += "https://"
			}

			authConfigurationKey += strings.Split(driverConfig.ImageName, "/")[0]
			if authConfiguration, ok := authConfigurations.Configs[authConfigurationKey]; ok {
				authOptions = authConfiguration
			}
		} else {
			return fmt.Errorf("Failed to open auth config file: %v, error: %v", authConfigFile, err)
		}
	}

	err := client.PullImage(pullOptions, authOptions)
	if err != nil {
		d.logger.Printf("[ERR] driver.docker: failed pulling container %s:%s: %s", repo, tag, err)
		return d.recoverablePullError(err, driverConfig.ImageName)
	}
	d.logger.Printf("[DEBUG] driver.docker: docker pull %s:%s succeeded", repo, tag)
	return nil
}

// loadImage creates an image by loading it from the file system
func (d *DockerDriver) loadImage(driverConfig *DockerDriverConfig, client *docker.Client, taskDir string) error {
	var errors multierror.Error
	for _, image := range driverConfig.LoadImages {
		archive := filepath.Join(taskDir, allocdir.TaskLocal, image)
		d.logger.Printf("[DEBUG] driver.docker: loading image from: %v", archive)
		f, err := os.Open(archive)
		if err != nil {
			errors.Errors = append(errors.Errors, fmt.Errorf("unable to open image archive: %v", err))
			continue
		}
		if err := client.LoadImage(docker.LoadImageOptions{InputStream: f}); err != nil {
			errors.Errors = append(errors.Errors, err)
		}
		f.Close()
	}
	return errors.ErrorOrNil()
}

func (d *DockerDriver) Start(ctx *ExecContext, task *structs.Task) (DriverHandle, error) {
	driverConfig, err := NewDockerDriverConfig(task)
	if err != nil {
		return nil, err
	}

	cleanupImage := d.config.ReadBoolDefault("docker.cleanup.image", true)

	taskDir, ok := ctx.AllocDir.TaskDirs[d.DriverContext.taskName]
	if !ok {
		return nil, fmt.Errorf("Could not find task directory for task: %v", d.DriverContext.taskName)
	}

	// Initialize docker API clients
	client, waitClient, err := d.dockerClients()
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to docker daemon: %s", err)
	}

	if err := d.createImage(driverConfig, client, taskDir); err != nil {
		return nil, fmt.Errorf("failed to create image: %v", err)
	}

	image := driverConfig.ImageName
	// Now that we have the image we can get the image id
	dockerImage, err := client.InspectImage(image)
	if err != nil {
		d.logger.Printf("[ERR] driver.docker: failed getting image id for %s: %s", image, err)
		return nil, fmt.Errorf("Failed to determine image id for `%s`: %s", image, err)
	}
	d.logger.Printf("[DEBUG] driver.docker: identified image %s as %s", image, dockerImage.ID)

	bin, err := discover.NomadExecutable()
	if err != nil {
		return nil, fmt.Errorf("unable to find the nomad binary: %v", err)
	}
	pluginLogFile := filepath.Join(taskDir, fmt.Sprintf("%s-executor.out", task.Name))
	pluginConfig := &plugin.ClientConfig{
		Cmd: exec.Command(bin, "executor", pluginLogFile),
	}

	exec, pluginClient, err := createExecutor(pluginConfig, d.config.LogOutput, d.config)
	if err != nil {
		return nil, err
	}
	executorCtx := &executor.ExecutorContext{
		TaskEnv:        d.taskEnv,
		Task:           task,
		Driver:         "docker",
		AllocDir:       ctx.AllocDir,
		AllocID:        ctx.AllocID,
		PortLowerBound: d.config.ClientMinPort,
		PortUpperBound: d.config.ClientMaxPort,
	}
	ss, err := exec.LaunchSyslogServer(executorCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to start syslog collector: %v", err)
	}

	config, err := d.createContainer(ctx, task, driverConfig, ss.Addr)
	if err != nil {
		d.logger.Printf("[ERR] driver.docker: failed to create container configuration for image %s: %s", image, err)
		pluginClient.Kill()
		return nil, fmt.Errorf("Failed to create container configuration for image %s: %s", image, err)
	}
	// Create a container
	container, err := client.CreateContainer(config)
	if err != nil {
		// If the container already exists because of a previous failure we'll
		// try to purge it and re-create it.
		if strings.Contains(err.Error(), "container already exists") {
			// Get the ID of the existing container so we can delete it
			containers, err := client.ListContainers(docker.ListContainersOptions{
				// The image might be in use by a stopped container, so check everything
				All: true,
				Filters: map[string][]string{
					"name": []string{config.Name},
				},
			})
			if err != nil {
				d.logger.Printf("[ERR] driver.docker: failed to query list of containers matching name:%s", config.Name)
				pluginClient.Kill()
				return nil, fmt.Errorf("Failed to query list of containers: %s", err)
			}

			// Couldn't find any matching containers
			if len(containers) == 0 {
				d.logger.Printf("[ERR] driver.docker: failed to get id for container %s: %#v", config.Name, containers)
				pluginClient.Kill()
				return nil, fmt.Errorf("Failed to get id for container %s", config.Name)
			}

			// Delete matching containers
			d.logger.Printf("[INFO] driver.docker: a container with the name %s already exists; will attempt to purge and re-create", config.Name)
			for _, container := range containers {
				err = client.RemoveContainer(docker.RemoveContainerOptions{
					ID: container.ID,
				})
				if err != nil {
					d.logger.Printf("[ERR] driver.docker: failed to purge container %s", container.ID)
					pluginClient.Kill()
					return nil, fmt.Errorf("Failed to purge container %s: %s", container.ID, err)
				}
				d.logger.Printf("[INFO] driver.docker: purged container %s", container.ID)
			}

			container, err = client.CreateContainer(config)
			if err != nil {
				d.logger.Printf("[ERR] driver.docker: failed to re-create container %s; aborting", config.Name)
				pluginClient.Kill()
				return nil, fmt.Errorf("Failed to re-create container %s; aborting", config.Name)
			}
		} else {
			// We failed to create the container for some other reason.
			d.logger.Printf("[ERR] driver.docker: failed to create container from image %s: %s", image, err)
			pluginClient.Kill()
			return nil, fmt.Errorf("Failed to create container from image %s: %s", image, err)
		}
	}
	d.logger.Printf("[INFO] driver.docker: created container %s", container.ID)

	// Start the container
	err = client.StartContainer(container.ID, container.HostConfig)
	if err != nil {
		d.logger.Printf("[ERR] driver.docker: failed to start container %s: %s", container.ID, err)
		pluginClient.Kill()
		return nil, fmt.Errorf("Failed to start container %s: %s", container.ID, err)
	}
	d.logger.Printf("[INFO] driver.docker: started container %s", container.ID)

	// Return a driver handle
	maxKill := d.DriverContext.config.MaxKillTimeout
	h := &DockerHandle{
		client:         client,
		waitClient:     waitClient,
		executor:       exec,
		pluginClient:   pluginClient,
		cleanupImage:   cleanupImage,
		logger:         d.logger,
		imageID:        dockerImage.ID,
		containerID:    container.ID,
		version:        d.config.Version,
		killTimeout:    GetKillTimeout(task.KillTimeout, maxKill),
		maxKillTimeout: maxKill,
		doneCh:         make(chan bool),
		waitCh:         make(chan *dstructs.WaitResult, 1),
	}
	if err := exec.SyncServices(consulContext(d.config, container.ID)); err != nil {
		d.logger.Printf("[ERR] driver.docker: error registering services with consul for task: %q: %v", task.Name, err)
	}
	go h.collectStats()
	go h.run()
	return h, nil
}

func (d *DockerDriver) Open(ctx *ExecContext, handleID string) (DriverHandle, error) {
	cleanupImage := d.config.ReadBoolDefault("docker.cleanup.image", true)

	// Split the handle
	pidBytes := []byte(strings.TrimPrefix(handleID, "DOCKER:"))
	pid := &dockerPID{}
	if err := json.Unmarshal(pidBytes, pid); err != nil {
		return nil, fmt.Errorf("Failed to parse handle '%s': %v", handleID, err)
	}
	d.logger.Printf("[INFO] driver.docker: re-attaching to docker process: %s", pid.ContainerID)
	d.logger.Printf("[DEBUG] driver.docker: re-attached to handle: %s", handleID)
	pluginConfig := &plugin.ClientConfig{
		Reattach: pid.PluginConfig.PluginConfig(),
	}

	client, waitClient, err := d.dockerClients()
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to docker daemon: %s", err)
	}

	// Look for a running container with this ID
	containers, err := client.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"id": []string{pid.ContainerID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to query for container %s: %v", pid.ContainerID, err)
	}

	found := false
	for _, container := range containers {
		if container.ID == pid.ContainerID {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("Failed to find container %s", pid.ContainerID)
	}
	exec, pluginClient, err := createExecutor(pluginConfig, d.config.LogOutput, d.config)
	if err != nil {
		d.logger.Printf("[INFO] driver.docker: couldn't re-attach to the plugin process: %v", err)
		d.logger.Printf("[DEBUG] driver.docker: stopping container %q", pid.ContainerID)
		if e := client.StopContainer(pid.ContainerID, uint(pid.KillTimeout.Seconds())); e != nil {
			d.logger.Printf("[DEBUG] driver.docker: couldn't stop container: %v", e)
		}
		return nil, err
	}

	ver, _ := exec.Version()
	d.logger.Printf("[DEBUG] driver.docker: version of executor: %v", ver.Version)

	// Return a driver handle
	h := &DockerHandle{
		client:         client,
		waitClient:     waitClient,
		executor:       exec,
		pluginClient:   pluginClient,
		cleanupImage:   cleanupImage,
		logger:         d.logger,
		imageID:        pid.ImageID,
		containerID:    pid.ContainerID,
		version:        pid.Version,
		killTimeout:    pid.KillTimeout,
		maxKillTimeout: pid.MaxKillTimeout,
		doneCh:         make(chan bool),
		waitCh:         make(chan *dstructs.WaitResult, 1),
	}
	if err := exec.SyncServices(consulContext(d.config, pid.ContainerID)); err != nil {
		h.logger.Printf("[ERR] driver.docker: error registering services with consul: %v", err)
	}

	go h.collectStats()
	go h.run()
	return h, nil
}

func (h *DockerHandle) ID() string {
	// Return a handle to the PID
	pid := dockerPID{
		Version:        h.version,
		ImageID:        h.imageID,
		ContainerID:    h.containerID,
		KillTimeout:    h.killTimeout,
		MaxKillTimeout: h.maxKillTimeout,
		PluginConfig:   NewPluginReattachConfig(h.pluginClient.ReattachConfig()),
	}
	data, err := json.Marshal(pid)
	if err != nil {
		h.logger.Printf("[ERR] driver.docker: failed to marshal docker PID to JSON: %s", err)
	}
	return fmt.Sprintf("DOCKER:%s", string(data))
}

func (h *DockerHandle) ContainerID() string {
	return h.containerID
}

func (h *DockerHandle) WaitCh() chan *dstructs.WaitResult {
	return h.waitCh
}

func (h *DockerHandle) Update(task *structs.Task) error {
	// Store the updated kill timeout.
	h.killTimeout = GetKillTimeout(task.KillTimeout, h.maxKillTimeout)
	if err := h.executor.UpdateTask(task); err != nil {
		h.logger.Printf("[DEBUG] driver.docker: failed to update log config: %v", err)
	}

	// Update is not possible
	return nil
}

// Kill is used to terminate the task. This uses `docker stop -t killTimeout`
func (h *DockerHandle) Kill() error {
	// Stop the container
	err := h.client.StopContainer(h.containerID, uint(h.killTimeout.Seconds()))
	if err != nil {
		h.executor.Exit()
		h.pluginClient.Kill()

		// Container has already been removed.
		if strings.Contains(err.Error(), NoSuchContainerError) {
			h.logger.Printf("[DEBUG] driver.docker: attempted to stop non-existent container %s", h.containerID)
			return nil
		}
		h.logger.Printf("[ERR] driver.docker: failed to stop container %s: %v", h.containerID, err)
		return fmt.Errorf("Failed to stop container %s: %s", h.containerID, err)
	}
	h.logger.Printf("[INFO] driver.docker: stopped container %s", h.containerID)
	return nil
}

func (h *DockerHandle) Stats() (*cstructs.TaskResourceUsage, error) {
	h.resourceUsageLock.RLock()
	defer h.resourceUsageLock.RUnlock()
	var err error
	if h.resourceUsage == nil {
		err = fmt.Errorf("stats collection hasn't started yet")
	}
	return h.resourceUsage, err
}

func (h *DockerHandle) run() {
	// Wait for it...
	exitCode, err := h.waitClient.WaitContainer(h.containerID)
	if err != nil {
		h.logger.Printf("[ERR] driver.docker: failed to wait for %s; container already terminated", h.containerID)
	}

	if exitCode != 0 {
		err = fmt.Errorf("Docker container exited with non-zero exit code: %d", exitCode)
	}

	close(h.doneCh)
	h.waitCh <- dstructs.NewWaitResult(exitCode, 0, err)
	close(h.waitCh)

	// Remove services
	if err := h.executor.DeregisterServices(); err != nil {
		h.logger.Printf("[ERR] driver.docker: error deregistering services: %v", err)
	}

	// Shutdown the syslog collector
	if err := h.executor.Exit(); err != nil {
		h.logger.Printf("[ERR] driver.docker: failed to kill the syslog collector: %v", err)
	}
	h.pluginClient.Kill()

	// Stop the container just incase the docker daemon's wait returned
	// incorrectly
	if err := h.client.StopContainer(h.containerID, 0); err != nil {
		_, noSuchContainer := err.(*docker.NoSuchContainer)
		_, containerNotRunning := err.(*docker.ContainerNotRunning)
		if !containerNotRunning && !noSuchContainer {
			h.logger.Printf("[ERR] driver.docker: error stopping container: %v", err)
		}
	}

	// Remove the container
	if err := h.client.RemoveContainer(docker.RemoveContainerOptions{ID: h.containerID, RemoveVolumes: true, Force: true}); err != nil {
		h.logger.Printf("[ERR] driver.docker: error removing container: %v", err)
	}

	// Cleanup the image
	if h.cleanupImage {
		if err := h.client.RemoveImage(h.imageID); err != nil {
			h.logger.Printf("[DEBUG] driver.docker: error removing image: %v", err)
		}
	}
}

// collectStats starts collecting resource usage stats of a docker container
func (h *DockerHandle) collectStats() {
	statsCh := make(chan *docker.Stats)
	statsOpts := docker.StatsOptions{ID: h.containerID, Done: h.doneCh, Stats: statsCh, Stream: true}
	go func() {
		//TODO handle Stats error
		if err := h.waitClient.Stats(statsOpts); err != nil {
			h.logger.Printf("[DEBUG] driver.docker: error collecting stats from container %s: %v", h.containerID, err)
		}
	}()
	numCores := runtime.NumCPU()
	for {
		select {
		case s := <-statsCh:
			if s != nil {
				ms := &cstructs.MemoryStats{
					RSS:      s.MemoryStats.Stats.Rss,
					Cache:    s.MemoryStats.Stats.Cache,
					Swap:     s.MemoryStats.Stats.Swap,
					MaxUsage: s.MemoryStats.MaxUsage,
					Measured: DockerMeasuredMemStats,
				}

				cs := &cstructs.CpuStats{
					ThrottledPeriods: s.CPUStats.ThrottlingData.ThrottledPeriods,
					ThrottledTime:    s.CPUStats.ThrottlingData.ThrottledTime,
					Measured:         DockerMeasuredCpuStats,
				}

				// Calculate percentage
				cores := len(s.CPUStats.CPUUsage.PercpuUsage)
				cs.Percent = calculatePercent(
					s.CPUStats.CPUUsage.TotalUsage, s.PreCPUStats.CPUUsage.TotalUsage,
					s.CPUStats.SystemCPUUsage, s.PreCPUStats.SystemCPUUsage, cores)
				cs.SystemMode = calculatePercent(
					s.CPUStats.CPUUsage.UsageInKernelmode, s.PreCPUStats.CPUUsage.UsageInKernelmode,
					s.CPUStats.CPUUsage.TotalUsage, s.PreCPUStats.CPUUsage.TotalUsage, cores)
				cs.UserMode = calculatePercent(
					s.CPUStats.CPUUsage.UsageInUsermode, s.PreCPUStats.CPUUsage.UsageInUsermode,
					s.CPUStats.CPUUsage.TotalUsage, s.PreCPUStats.CPUUsage.TotalUsage, cores)
				cs.TotalTicks = (cs.Percent / 100) * shelpers.TotalTicksAvailable() / float64(numCores)

				h.resourceUsageLock.Lock()
				h.resourceUsage = &cstructs.TaskResourceUsage{
					ResourceUsage: &cstructs.ResourceUsage{
						MemoryStats: ms,
						CpuStats:    cs,
					},
					Timestamp: s.Read.UTC().UnixNano(),
				}
				h.resourceUsageLock.Unlock()
			}
		case <-h.doneCh:
			return
		}
	}
}

func calculatePercent(newSample, oldSample, newTotal, oldTotal uint64, cores int) float64 {
	numerator := newSample - oldSample
	denom := newTotal - oldTotal
	if numerator <= 0 || denom <= 0 {
		return 0.0
	}

	return (float64(numerator) / float64(denom)) * float64(cores) * 100.0
}
