package factory

import "github.com/go-season/ginctl/pkg/util/log"

type Factory interface {
	GetLog() log.Logger
}

type DefaultFactoryImpl struct{}

func DefaultFactory() Factory {
	return &DefaultFactoryImpl{}
}

func (f *DefaultFactoryImpl) GetLog() log.Logger {
	return log.GetInstance()
}
