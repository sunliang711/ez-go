package ezcfg

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type ConfigFile struct {
	Path     string
	Required bool
}

type ConfigManager struct {
	configFiles []ConfigFile
	envPrefix   string
	logger      *log.Logger
}

// envPrefix: 环境变量前缀 环境变量需要全部大写，用下划线分隔，如：APP_NAME,如果没有前缀，则也不需要前导下划线,如: NAME
// configFiles: 配置文件列表，按顺序读取，后面的配置文件会覆盖前面的配置文件
func NewConfigManager(envPrefix string, configFiles []ConfigFile) *ConfigManager {
	manager := &ConfigManager{
		configFiles: configFiles,
		envPrefix:   envPrefix,
		logger:      log.New(os.Stdout, "|ConfigManager| ", log.LstdFlags),
	}

	return manager
}

func (m *ConfigManager) Init(configuration any) error {
	configFiles := []string{}
	for _, file := range m.configFiles {
		if file.Required {
			if !fileExists(file.Path) {
				return fmt.Errorf("config file: %s not found", file.Path)
			}
			m.logger.Printf("add config file: %s", file.Path)
			configFiles = append(configFiles, file.Path)
		} else {
			if fileExists(file.Path) {
				configFiles = append(configFiles, file.Path)
				m.logger.Printf("add config file: %s", file.Path)
			} else {
				m.logger.Printf("skip unrequired config file: %s", file.Path)
			}
		}
	}

	m.logger.Printf("load configuration from config files: %v and env var with prefix: %s", configFiles, m.envPrefix)

	err := loadConfig(m.envPrefix, configFiles, configuration)
	if err != nil {
		return err
	}

	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// loadConfig 加载并合并多个配置文件和环境变量到给定的结构体中
// paths中的配置文件按顺序读取，后面的配置文件会覆盖前面的配置文件
// 文件格式可以是json、yaml、toml
func loadConfig(envPrefix string, files []string, config interface{}) error {
	v := viper.New()

	// 设置环境变量前缀
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取并合并配置文件
	for _, file := range files {
		v.SetConfigFile(file)
		if err := v.MergeInConfig(); err != nil {
			log.Printf("Error reading config file %s: %v", file, err)
		}
	}

	// 反序列化配置到结构体
	if err := v.Unmarshal(config); err != nil {
		return err
	}

	return nil
}
