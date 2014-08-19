package main

// ConfigFile returns the default path to the configuration file. On
// Unix-like systems this is the ".terraformrc" file in the home directory.
// On Windows, this is the "terraform.rc" file in the application data
// directory.
func ConfigFile() (string, error) {
    return configFile()
}
