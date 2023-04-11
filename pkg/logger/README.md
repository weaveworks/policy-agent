# Logger
- a wrapper over `uber/zap` logger
- There are 4 levels that can be used; info, error, debug, and warning.


## Usage
It's usually not ideal to pass `logger` instance across you application, this can be seen as unnecessary pollution. That's why **global** logger is provided by default with `Info` as its default log level.

```go
import "github.com/MagalixTechnologies/core/logger"

logger.Info("this is a log")
```

### Change log level
To change the log level of the global logger you can simply use the `Config` function

```go
import "github.com/MagalixTechnologies/core/logger"

logger.Config(logger.DebugLevel)
logger.Debug("this is a debug log")
```

### Create Custom logger
sometimes a custom logger is ideal. For example creating a logger that have `request-id` on all the logs. you can create custom logger by using `With` or `New` function

```go
import "github.com/MagalixTechnologies/core/logger"
// using with will have the global logger log level
customLogger := logger.With("requestId", reqID)
customLogger.Info("this is a debug log")

l := logger.New(logger.DebugLevel)
l.Debug("this is a debug log")
```
