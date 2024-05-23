package registry

import "github.com/hibiken/asynq"

// TaskRegistry is the default registry for tasks.
var TaskRegistry = New[string, asynq.Handler]()
