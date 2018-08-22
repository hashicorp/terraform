package clientconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
)

// defaultIfEmpty is a helper function to make it cleaner to set default value
// for strings.
func defaultIfEmpty(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// updateAuthInfo updates AuthInfo in Cloud struct with AuthInfo in PublicCloud.
func updateAuthInfo(cloud *Cloud, publicCloud *PublicCloud) {
	s := reflect.ValueOf(publicCloud.AuthInfo).Elem()
	for i := 0; i < s.NumField(); i++ {
		v := s.Field(i).Interface().(string)
		if v != "" {
			reflect.ValueOf(cloud.AuthInfo).Elem().Field(i).SetString(v)
		}
	}
}

// findAndReadCloudsYAML attempts to locate a clouds.yaml file in the following
// locations:
//
// 1. OS_CLIENT_CONFIG_FILE
// 2. Current directory.
// 3. unix-specific user_config_dir (~/.config/openstack/clouds.yaml)
// 4. unix-specific site_config_dir (/etc/openstack/clouds.yaml)
//
// If found, the contents of the file is returned.
func findAndReadCloudsYAML() ([]byte, error) {
	// OS_CLIENT_CONFIG_FILE
	if v := os.Getenv("OS_CLIENT_CONFIG_FILE"); v != "" {
		if ok := fileExists(v); ok {
			return ioutil.ReadFile(v)
		}
	}

	return findAndReadYAML("clouds.yaml")
}

func findAndReadPublicCloudsYAML() ([]byte, error) {
	return findAndReadYAML("clouds-public.yaml")
}

func findAndReadYAML(yamlFile string) ([]byte, error) {
	// current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("unable to determine working directory: %s", err)
	}

	filename := filepath.Join(cwd, yamlFile)
	if ok := fileExists(filename); ok {
		return ioutil.ReadFile(filename)
	}

	// unix user config directory: ~/.config/openstack.
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to get current user: %s", err)
	}

	homeDir := currentUser.HomeDir
	if homeDir != "" {
		filename := filepath.Join(homeDir, ".config/openstack/"+yamlFile)
		if ok := fileExists(filename); ok {
			return ioutil.ReadFile(filename)
		}
	}

	// unix-specific site config directory: /etc/openstack.
	if ok := fileExists("/etc/openstack/" + yamlFile); ok {
		return ioutil.ReadFile("/etc/openstack/" + yamlFile)
	}

	return nil, fmt.Errorf("no " + yamlFile + " file found")
}

// fileExists checks for the existence of a file at a given location.
func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}
