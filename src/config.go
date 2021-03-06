package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jinzhu/configor"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

/**
 * ConfigurationLoader contains all methods to load/save configuration files
 */
type ConfigurationLoader struct {
}

/**
 * The Project Configuration
 */
type ProjectConfigrationFile struct {
	Images []struct {
		// name of the container
		Name string

		// description for the  container
		Description string

		// the commands provided by the image
		Provides []string

		// container image
		Image string

		// target directory to mount your project inside of the container
		Directory string `default:"/project"`

		// wrap the executed command inside of the container into a shell (ex. if you use globs)
		Shell string `default:"none"`

		// commands that should run in the container before the actual command is executed
		BeforeScript []string `yaml:"before_script"`

		// Caching of container-directories
		Caching []CachingEntry `yaml:"cache"`

		// the command scope (internal use only) - global or project
		Scope string
	}
}

type CachingEntry struct {

	/**
	 * Name of the caching entry
	 */
	Name string `yaml:"name",default:""`

	/**
	 * Directory inside of the container that should be mounted on the host within the cache directory
	 */
	ContainerDirectory string `yaml:"directory",default:""`
}

/**
 * The EnvCLI Configuration
 */
type PropertyConfigurationFile struct {
	Properties map[string]string
}

/**
 * Load the project config
 */
func (configurationLoader ConfigurationLoader) loadProjectConfig(configFile string) (ProjectConfigrationFile, error) {
	var cfg ProjectConfigrationFile

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Debugf("Can't load config - file [%s] does not exist!", configFile)
		return ProjectConfigrationFile{}, nil
	}

	log.Debugf("Loading project configuration file %s", configFile)
	configor.New(&configor.Config{Debug: false}).Load(&cfg, configFile)

	return cfg, nil
}

/**
 * Load the property config
 */
func (configurationLoader ConfigurationLoader) loadPropertyConfig(configFile string) (PropertyConfigurationFile, error) {
	var cfg PropertyConfigurationFile
	cfg.Properties = make(map[string]string)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Debugf("Can't load global properties - file [%s] does not exist!", configFile)
		return cfg, nil
	}

	log.Debug("Loading property configuration file " + configFile)
	configor.New(&configor.Config{Debug: false}).Load(&cfg, configFile)

	return cfg, nil
}

/**
 * Save the global config
 */
func (configurationLoader ConfigurationLoader) savePropertyConfig(configFile string, cfg PropertyConfigurationFile) error {
	log.Debug("Saving property configuration file " + configFile)

	fileContent, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(configFile, fileContent, 0600)
}

/**
 * Get the execution directory
 */
func (configurationLoader ConfigurationLoader) getExecutionDirectory() string {
	ex, err := os.Executable()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Couldn't detect execution directory!")
		return ""
	}

	return filepath.Dir(ex)
}

/**
 * Get the project root directory by searching for the envcli config
 */
func (configurationLoader ConfigurationLoader) getProjectDirectory() string {
	log.WithFields(log.Fields{
		"method": "getProjectDirectory()",
	}).Debugf("Trying to detect project directory ...")

	currentDirectory := getWorkingDirectory()
	var projectDirectory = ""
	log.WithFields(log.Fields{
		"method": "getProjectDirectory()",
	}).Debugf("current working directory [%s]", currentDirectory)

	directoryParts := strings.Split(currentDirectory, string(os.PathSeparator))

	for projectDirectory == "" {
		if _, err := os.Stat(filepath.Join(currentDirectory, "/.envcli.yml")); err == nil {
			log.WithFields(log.Fields{
				"method": "getProjectDirectory()",
			}).Debugf("found project config in directory [%s]", currentDirectory)
			return currentDirectory
		}

		if directoryParts[0]+"\\" == currentDirectory || currentDirectory == "/" {
			log.WithFields(log.Fields{
				"method": "getProjectDirectory()",
			}).Debugf("didn't find a envcli project config in any parent directors")
			return ""
		}

		currentDirectory = filepath.Dir(currentDirectory)
		log.WithFields(log.Fields{
			"method": "getProjectDirectory()",
		}).Debugf("proceed to search next directory [%s]", currentDirectory)
	}

	return ""
}

/**
 * Merge two configurations and keep the origin in the Scope
 * TODO: Handle conflicts with a warning / by order project definition have precedence right now
 */
func (configurationLoader ConfigurationLoader) mergeConfigurations(configProject ProjectConfigrationFile, configGlobal ProjectConfigrationFile) ProjectConfigrationFile {
	var cfg = ProjectConfigrationFile{}

	for _, image := range configProject.Images {
		image.Scope = "Project"
		cfg.Images = append(cfg.Images, image)
	}
	for _, image := range configGlobal.Images {
		image.Scope = "Global"
		cfg.Images = append(cfg.Images, image)
	}

	return cfg
}
