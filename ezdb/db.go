package ezdb

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

type Database struct {
	dbs     map[string]*gorm.DB
	configs map[string]*DatabaseConfig
	logger  *log.Logger
}

const (
	DB_MYSQL    = "mysql"
	DB_SQLITE   = "sqlite"
	DB_POSTGRES = "postgres"
)

const (
	DBLOG_SILENT = "silent"
	DBLOG_INFO   = "info"
	DBLOG_ERROR  = "error"
	DBLOG_WARN   = "warn"
)

var opens = map[string]func(string) gorm.Dialector{
	DB_MYSQL:    mysql.Open,
	DB_SQLITE:   sqlite.Open,
	DB_POSTGRES: postgres.Open,
}

type DatabaseConfig struct {
	Name   string
	Dsn    string
	Driver string
	Tables []Table
	Log    string // silent, info, warn, error
}

func (cfg *DatabaseConfig) Check() error {
	switch cfg.Driver {
	case DB_MYSQL:
	case DB_SQLITE:
	case DB_POSTGRES:
	default:
		return fmt.Errorf("check database config with driver: %s error: %w", cfg.Driver, ErrInvalidDBDriver)
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
		db.AddDatabase(config)
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
func (db *Database) AddDatabase(config *DatabaseConfig) {
	db.configs[config.Name] = config
}

// Init 初始化数据库连接
func (db *Database) Init() error {
	if len(db.configs) == 0 {
		return fmt.Errorf("no database config")
	}

	for _, config := range db.configs {
		db.logger.Printf("open database: %s", config.Name)

		var logger glogger.Interface
		switch config.Log {
		case DBLOG_SILENT:
			logger = glogger.Default.LogMode(glogger.Silent)
		case DBLOG_ERROR:
			logger = glogger.Default.LogMode(glogger.Error)
		case DBLOG_WARN:
			logger = glogger.Default.LogMode(glogger.Warn)
		case DBLOG_INFO:
			logger = glogger.Default.LogMode(glogger.Info)

		default:
			logger = glogger.Default.LogMode(glogger.Silent)
		}
		conn, err := gorm.Open(opens[config.Driver](config.Dsn), &gorm.Config{Logger: logger})
		if err != nil {
			return fmt.Errorf("open database %s error: %v", config.Name, err)
		}
		db.dbs[config.Name] = conn

		// migrate tables
		for _, table := range config.Tables {
			db.logger.Printf("migrate table: %s", table.Name)
			err = conn.AutoMigrate(table.Definition)
			if err != nil {
				return fmt.Errorf("migrate table %s error: %v", table.Name, err)
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
