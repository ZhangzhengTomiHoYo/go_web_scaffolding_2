package logger

import (
	"go_web_scaffolding/settings"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 此时的就不能全局，否则在main里会是：logger.Logger.Debug()，变量会很长
// var Logger *zap.Logger

func Init(cfg *settings.LogConfig) (err error) {
	writeSyncer := getLogWriter(
		cfg.Filename,
		cfg.MaxSize,
		cfg.MaxBackups,
		cfg.MaxAge,
	)

	encoder := getEncoder()

	// 3. 解析日志级别（从配置字符串转 zap 识别的 Level 类型）
	// 声明一个 zapcore.Level 类型的指针变量 l
	// zapcore.Level 是 zap 定义的日志级别枚举类型（比如 InfoLevel、ErrorLevel、WarnLevel 等）
	// 用 new 关键字创建指针，因为 UnmarshalText 方法需要接收指针（修改变量本身的值）
	var level = new(zapcore.Level)

	// 关键说明：
	// 配置文件中 log.level 是字符串（比如 "info"、"error"、"warn"、"debug"），但 zap 不直接识别字符串
	// 需要通过 UnmarshalText 方法将字符串（转成 []byte）解析为 zapcore.Level 类型的值
	// []byte(viper.GetString("log.level"))：把配置里的字符串转成字节切片（UnmarshalText 要求的参数类型）
	err = level.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return
	}
	//
	// 将 1编码器 2写入器 3级别 组装成core
	core := zapcore.NewCore(encoder, writeSyncer, level)
	// New()是把核心零件组装成 完整的日志实例
	// 其中，zap.AddCaller()是让 zap 沿着「函数调用链」向上找，记录「直接调用日志方法（如 Info/Error）的那一行代码」的位置。
	lg := zap.New(core, zap.AddCaller())
	// zap.ReplaceGlobals(lg)
	// 核心作用：把自定义的日志实例设为「全局默认」，不用到处传参
	zap.ReplaceGlobals(lg)

	return
}

// getLogWriter 创建一个支持日志文件切割/备份的 zap 日志写入器
// 参数说明：
//
//	filename: 日志文件的保存路径+文件名（例如："./logs/app.log"）
//	maxSize: 单个日志文件的最大大小（单位：MB），超过则自动切割
//	maxBackup: 保留的日志备份文件最大数量，超出则删除最旧的
//	maxAge: 日志文件保留的最大天数，超出则自动删除
//
// 返回值：zapcore.WriteSyncer - zap 日志核心的写入器接口，用于将日志写入文件
func getLogWriter(filename string, maxSize, maxBackup, maxAge int) zapcore.WriteSyncer {
	// 初始化 lumberjack.Logger 实例（日志文件切割器）
	// lumberjack 是专门处理日志文件切割、备份、过期删除的工具
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,  // 设置日志文件的存储路径和名称
		MaxSize:    maxSize,   // 设置单个日志文件的最大大小（MB），比如设100表示文件到100MB就切割
		MaxBackups: maxBackup, // 设置保留的日志备份文件最大数量，比如设10表示最多保留10个备份文件
		MaxAge:     maxAge,    // 设置日志文件保留的最大天数，比如设7表示7天前的日志文件会被自动删除
	}

	// 将 lumberjack 的日志写入器适配为 zap 核心能识别的 WriteSyncer 接口
	// 作用：让 zap 日志库能通过这个写入器，将日志输出到配置好的切割文件中
	return zapcore.AddSync(lumberJackLogger)
}

// getEncoder 创建 zap 日志的 JSON 格式编码器（定义日志输出的格式规则）
// 返回值：zapcore.Encoder - zap 日志的编码器接口，控制日志的输出格式
func getEncoder() zapcore.Encoder {
	// 获取 zap 库提供的「生产环境默认编码器配置」
	// 这个默认配置包含了基础的日志字段（如级别、时间、调用者等），我们在此基础上自定义
	encoderConfig := zap.NewProductionEncoderConfig()

	// 配置日志中「时间字段」的编码格式：使用 ISO8601 标准格式（例如：2025-12-03T12:34:56.789Z）
	// 默认的时间格式是时间戳，换成 ISO8601 更易读
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 配置日志中「时间字段」的键名：将默认的 "ts" 改为更直观的 "time"
	// 比如原本日志里是 "ts": 1735921234，改后是 "time": "2025-12-03T12:34:56.789Z"
	encoderConfig.TimeKey = "time"

	// 配置日志「级别字段」的编码格式：使用大写字母（例如：INFO、ERROR、WARN）
	// 默认是小写（info/error），大写更符合日志阅读习惯
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 配置日志「耗时字段」的编码格式：将耗时转换为「秒」为单位的浮点数
	// 比如原本是纳秒级的数字，改后会显示成 "duration": 0.002（表示2毫秒）
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder

	// 配置日志「调用者字段」的编码格式：使用短格式（包名/文件名:行号）
	// 比如长格式是 "caller": "github.com/xxx/project/pkg/log/log.go:25"，短格式会简化为 "pkg/log/log.go:25"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 根据自定义的编码器配置，创建 JSON 格式的日志编码器
	// 最终日志会以 JSON 格式输出，例如：{"time":"2025-12-03T12:34:56.789Z","level":"INFO","caller":"pkg/log/log.go:25","msg":"日志内容"}
	return zapcore.NewJSONEncoder(encoderConfig)
}

// 使用zap接收gin框架日志
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		zap.L().Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}
}

// GinRecovery recover掉项目可能出现的panic，并使用zap记录相关日志
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					zap.L().Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					zap.L().Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					zap.L().Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
