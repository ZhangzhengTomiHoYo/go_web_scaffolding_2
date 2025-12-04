package settings

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Conf 全局变量，用来保存程序的所有配置信息
var Conf = new(AppConfig)

// viper的Tag
type AppConfig struct {
	Name         string `mapstructure:"name"`
	Mode         string `mapstructure:"mode"`
	Version      string `mapstructure:"version"`
	Port         int    `mapstructure:"port"`
	*LogConfig   `mapstructure:"log"`
	*MySQLConfig `mapstructure:"mysql"`
	*RedisConfig `mapstructure:"redis"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DbName       string `mapstructure:"db_name"`
	Port         int    `mapstructure:"port"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Password string `mapstructure:"password"`
	Port     int    `mapstructure:"port"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

func Init() (err error) {
	//viper.SetConfigFile("config.yaml") // 直接指定文件名称
	viper.SetConfigName("config") // 指定配置文件名称（不需要带后缀）
	viper.SetConfigType("yaml")   // 指定配置文件类型 (专用于远程，本地不生效)
	viper.AddConfigPath("./")     // 指定查找配置文件的路径
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	// 上面是一样的
	//
	// 使用结构体，需要将配置 反序列化到Conf变量中
	if err := viper.Unmarshal(Conf); err != nil {

	}
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件修改了...")
		// 当配置文件发生变化，再次反序列化到变量中
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("")
		}
	})
	return
}
