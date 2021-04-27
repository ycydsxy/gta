package gta

import (
	"fmt"
	"sync"
)

const (
	varchar64MaxLenth = 64
)

type taskRegister interface {
	Register(key TaskKey, def TaskDefinition) error
	GetDefinition(key TaskKey) (*TaskDefinition, error)
	GroupKeysByInitTimeoutSensitivity() ([]TaskKey, []TaskKey)
	GetBuiltInKeys() []TaskKey
}

type taskRegisterImp struct {
	defMap sync.Map
}

func (s *taskRegisterImp) Register(key TaskKey, def TaskDefinition) error {
	if len([]rune(key)) > varchar64MaxLenth {
		return fmt.Errorf("task_key exceed max length: %v", key)
	}
	if err := def.init(key); err != nil {
		return fmt.Errorf("definition validate error, task_key: %v, caused by: %w", key, err)
	}
	_, loaded := s.defMap.LoadOrStore(def.key, &def)
	if loaded {
		return fmt.Errorf("definition already registered, task_key: %v", def.key)
	}
	return nil
}

func (s *taskRegisterImp) GetDefinition(key TaskKey) (*TaskDefinition, error) {
	value, ok := s.defMap.Load(key)
	if !ok {
		return nil, fmt.Errorf("definition not found, task_key: %v", key) // TODO
	}
	return value.(*TaskDefinition), nil
}

func (s *taskRegisterImp) GroupKeysByInitTimeoutSensitivity() ([]TaskKey, []TaskKey) {
	sensitiveKeys, insensitiveKeys := make([]TaskKey, 0), make([]TaskKey, 0)
	s.defMap.Range(func(key, value interface{}) bool {
		if def := value.(*TaskDefinition); def.InitTimeoutSensitive {
			sensitiveKeys = append(sensitiveKeys, key.(TaskKey))
		} else {
			insensitiveKeys = append(insensitiveKeys, key.(TaskKey))
		}
		return true
	})
	return sensitiveKeys, insensitiveKeys
}

func (s *taskRegisterImp) GetBuiltInKeys() []TaskKey {
	res := make([]TaskKey, 0)
	s.defMap.Range(func(key, value interface{}) bool {
		if def := value.(*TaskDefinition); def.builtin {
			res = append(res, key.(TaskKey))
		}
		return true
	})
	return res
}
