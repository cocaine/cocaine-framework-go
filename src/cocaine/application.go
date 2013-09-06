package cocaine

import (
//"codec"
//"fmt"
)

type Application struct {
	*Service
}

//map[0:enqueue 1:info]
const (
	APPLICATION_ENQUEUE = iota
	APPLICATION_INFO
)

func NewApplication(name string, host string, port uint64) (*Application, error) {
	return &Application{NewService(host, port, name)}, nil
}

func (app *Application) Enqueue(event string, data []byte) chan ServiceResult {
	return app.Call(APPLICATION_ENQUEUE, event, data)
}

type ApplicationInfoRes struct {
	Res map[string]interface{}
	err error
}

func (app *Application) Info() chan ApplicationInfoRes {
	Out := make(chan ApplicationInfoRes)
	go func() {
		info := make(map[string]interface{})
		if res := <-app.Call(APPLICATION_INFO, []interface{}{}); res.err == nil {
			// Add some type assertations in future
			for k, v := range res.result.(map[interface{}]interface{}) {
				info[k.(string)] = v
			}
			Out <- ApplicationInfoRes{Res: info, err: nil}
		} else {
			Out <- ApplicationInfoRes{Res: nil, err: res.err}
		}
	}()
	return Out
}
