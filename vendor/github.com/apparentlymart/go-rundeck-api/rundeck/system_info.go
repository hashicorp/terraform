package rundeck

import (
	"encoding/xml"
	"time"
)

// SystemInfo represents a set of miscellaneous system information properties about the
// Rundeck server.
type SystemInfo struct {
	XMLName    xml.Name        `xml:"system"`
	ServerTime SystemTimestamp `xml:"timestamp"`
	Rundeck    About           `xml:"rundeck"`
	OS         SystemOS        `xml:"os"`
	JVM        SystemJVM       `xml:"jvm"`
	Stats      SystemStats     `xml:"stats"`
}

// About describes the Rundeck server itself.
type About struct {
	XMLName    xml.Name `xml:"rundeck"`
	Version    string   `xml:"version"`
	APIVersion int64    `xml:"apiversion"`
	Build      string   `xml:"build"`
	Node       string   `xml:"node"`
	BaseDir    string   `xml:"base"`
	ServerUUID string   `xml:"serverUUID,omitempty"`
}

// SystemTimestamp gives a timestamp from the Rundeck server.
type SystemTimestamp struct {
	Epoch       string `xml:"epoch,attr"`
	EpochUnit   string `xml:"unit,attr"`
	DateTimeStr string `xml:"datetime"`
}

// SystemOS describes the operating system of the Rundeck server.
type SystemOS struct {
	Architecture string `xml:"arch"`
	Name         string `xml:"name"`
	Version      string `xml:"version"`
}

// SystemJVM describes the Java Virtual Machine that the Rundeck server is running in.
type SystemJVM struct {
	Name                  string `xml:"name"`
	Vendor                string `xml:"vendor"`
	Version               string `xml:"version"`
	ImplementationVersion string `xml:"implementationVersion"`
}

// SystemStats provides some basic system statistics about the server that Rundeck is running on.
type SystemStats struct {
	XMLName   xml.Name             `xml:"stats"`
	Uptime    SystemUptime         `xml:"uptime"`
	CPU       SystemCPUStats       `xml:"cpu"`
	Memory    SystemMemoryUsage    `xml:"memory"`
	Scheduler SystemSchedulerStats `xml:"scheduler"`
	Threads   SystemThreadStats    `xml:"threads"`
}

// SystemUptime describes how long Rundeck's host machine has been running.
type SystemUptime struct {
	XMLName       xml.Name        `xml:"uptime"`
	Duration      string          `xml:"duration,attr"`
	DurationUnit  string          `xml:"unit,attr"`
	BootTimestamp SystemTimestamp `xml:"since"`
}

// SystemCPUStats describes the available processors and the system load average of the machine on
// which the Rundeck server is running.
type SystemCPUStats struct {
	XMLName     xml.Name `xml:"cpu"`
	LoadAverage struct {
		Unit  string  `xml:"unit,attr"`
		Value float64 `xml:",chardata"`
	} `xml:"loadAverage"`
	ProcessorCount int64 `xml:"processors"`
}

// SystemMemoryUsage describes how much memory is available and used on the machine on which
// the Rundeck server is running.
type SystemMemoryUsage struct {
	XMLName xml.Name `xml:"memory"`
	Unit    string   `xml:"unit,attr"`
	Max     int64    `xml:"max"`
	Free    int64    `xml:"free"`
	Total   int64    `xml:"total"`
}

// SystemSchedulerStats provides statistics about the Rundeck scheduler.
type SystemSchedulerStats struct {
	RunningJobCount int64 `xml:"running"`
}

// SystemThreadStats provides statistics about the thread usage of the Rundeck server.
type SystemThreadStats struct {
	ActiveThreadCount int64 `xml:"active"`
}

// GetSystemInfo retrieves and returns miscellaneous system information about the Rundeck server
// and the machine it's running on.
func (c *Client) GetSystemInfo() (*SystemInfo, error) {
	sysInfo := &SystemInfo{}
	err := c.get([]string{"system", "info"}, nil, sysInfo)
	return sysInfo, err
}

// DateTime produces a time.Time object from a SystemTimestamp object.
func (ts *SystemTimestamp) DateTime() time.Time {
	// Assume the server will always give us a valid timestamp,
	// so we don't need to handle the error case.
	// (Famous last words?)
	t, _ := time.Parse(time.RFC3339, ts.DateTimeStr)
	return t
}
