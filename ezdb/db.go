package ezdb

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type Database struct {
	dbs     map[string]*gorm.DB
	configs map[string]*DatabaseConfig
	logger  *log.Logger
}

type Driver int8

func (d *Driver) String() string {
	switch *d {
	case DRIVER_MYSQL:
		return "mysql"
	case DRIVER_SQLITE:
		return "sqlite"
	case DRIVER_POSTGRES:
		return "postgres"
	default:
		return "unknown"
	}
}

const (
	DRIVER_MYSQL    Driver = iota // MySQL
	DRIVER_SQLITE                 // SQLite
	DRIVER_POSTGRES               // PostgreSQL
)

type LogLevel int8

const (
	LOG_LEVEL_SILENT LogLevel = iota // Silent
	LOG_LEVEL_INFO                   // Info
	LOG_LEVEL_ERROR                  // Error
	LOG_LEVEL_WARN                   // Warn
)

type Policy int8

const (
	Random           Policy = iota // Random
	RoundRobin                     // Round Robin
	StrictRoundRobin               // Strict Round Robin
)

var opens = map[Driver]func(string) gorm.Dialector{
	DRIVER_MYSQL:    mysql.Open,
	DRIVER_SQLITE:   sqlite.Open,
	DRIVER_POSTGRES: postgres.Open,
}

type DatabaseConfig struct {
	Name   string
	Dsn    string
	Driver Driver
	Tables []Table
	Log    LogLevel

	AutoMigrate bool

	// ReadWriteSeparate 是否开启读写分离
	ReadWriteSeparate bool
	// Replicas 读写分离时的从库地址，多个地址用||分隔
	Replicas string
	Policy   Policy // 读写分离策略，默认随机
}

func (cfg *DatabaseConfig) Check() error {
	switch cfg.Driver {
	case DRIVER_MYSQL:
	case DRIVER_SQLITE:
	case DRIVER_POSTGRES:
	default:
		return fmt.Errorf("check database config with driver: %v error: %w", cfg.Driver, ErrInvalidDBDriver)
	}
	return nil
}

func NewDatabase(configs []*DatabaseConfig, init bool) (*Database, error) {
	db := &Database{
		dbs:     make(map[string]*gorm.DB),
		configs: make(map[string]*DatabaseConfig),
		logger:  log.New(os.Stdout, "|Database| ", log.LstdFlags),
	}

	for _, config := range configs {
		err := config.Check()
		if err != nil {
			return nil, err
		}

		if err := db.AddDatabase(config); err != nil {
			return nil, err
		}
	}

	if init {
		err := db.Init()
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

// AddDatabase 增加数据库配置
func (db *Database) AddDatabase(config *DatabaseConfig) error {
	// 检查是否已经存在同名的配置
	if _, exists := db.configs[config.Name]; exists {
		return fmt.Errorf("duplicate database config name: %s", config.Name)
	}
	db.configs[config.Name] = config

	return nil
}

// GetDatabaseConfig
func (db *Database) GetDatabaseConfig(name string) *DatabaseConfig {
	return db.configs[name]
}

// AddTable 增加表
func (db *Database) AddTable(name string, table Table) {
	if cfg, ok := db.configs[name]; ok {
		cfg.Tables = append(cfg.Tables, table)
	}
}

// Init 初始化数据库连接
func (db *Database) Init() error {
	if len(db.configs) == 0 {
		return fmt.Errorf("no database config")
	}

	for _, config := range db.configs {
		db.logger.Printf("open database: %s", config.Name)

		driver := opens[config.Driver]

		var logger glogger.Interface
		switch config.Log {
		case LOG_LEVEL_SILENT:
			logger = glogger.Default.LogMode(glogger.Silent)
		case LOG_LEVEL_ERROR:
			logger = glogger.Default.LogMode(glogger.Error)
		case LOG_LEVEL_WARN:
			logger = glogger.Default.LogMode(glogger.Warn)
		case LOG_LEVEL_INFO:
			logger = glogger.Default.LogMode(glogger.Info)

		default:
			logger = glogger.Default.LogMode(glogger.Silent)
		}
		conn, err := gorm.Open(driver(config.Dsn), &gorm.Config{Logger: logger})
		if err != nil {
			return fmt.Errorf("open database %s error: %v", config.Name, err)
		}

		// 读写分离
		var replicas []gorm.Dialector
		if config.ReadWriteSeparate && config.Replicas != "" {
			// parse replicas
			replicaList := strings.Split(config.Replicas, "||")
			if len(replicaList) == 0 {
				return fmt.Errorf("read write separate replicas is empty for database %s", config.Name)
			}
			var policy dbresolver.Policy
			switch config.Policy {
			case Random:
				policy = dbresolver.RandomPolicy{}
			case RoundRobin:
				policy = dbresolver.RoundRobinPolicy()
			case StrictRoundRobin:
				policy = dbresolver.StrictRoundRobinPolicy()
			default:
				return fmt.Errorf("invalid read write separate policy: %v for database %s", config.Policy, config.Name)
			}

			// open replicas
			for _, r := range replicaList {
				if len(r) == 0 {
					continue // 跳过空地址
				}
				db.logger.Printf("open replica database: %s", r)
				replicas = append(replicas, driver(r))
			}

			if len(replicas) > 0 {
				dbresolverConfig := dbresolver.Config{
					Sources:  []gorm.Dialector{driver(config.Dsn)},
					Replicas: replicas,
					Policy:   policy,
				}
				readWritePlugin := dbresolver.Register(dbresolverConfig)
				if err := conn.Use(readWritePlugin); err != nil {
					return fmt.Errorf("use read write plugin error: %v", err)
				}
			}
		}

		db.logger.Printf("open database %s success", config.Name)
		db.dbs[config.Name] = conn

		if config.AutoMigrate {
			// migrate tables
			for _, table := range config.Tables {
				db.logger.Printf("migrate table: %s", table.Name)
				err = conn.AutoMigrate(table.Definition)
				if err != nil {
					return fmt.Errorf("migrate table %s error: %v", table.Name, err)
				}
			}
		}
	}

	return nil
}

// GetDatabase 获取数据库连接
func (db *Database) GetDB(name string) *gorm.DB {
	return db.dbs[name]
}

type Table struct {
	Name       string
	Definition any // model
}

func (db *Database) Close() error {
	for name, conn := range db.dbs {
		db.logger.Printf("close database: %s", name)
		sqlDB, err := conn.DB()
		if err != nil {
			return err
		}
		err = sqlDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
