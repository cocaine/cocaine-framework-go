package cocaineapi

import (
	"github.com/cocaine/cocaine-framework-go/cocaine"
)

type Runlist map[string]string
type LaunchResult map[string]string

type Node interface {
	AppList() ([]string, error)
	StartApp(Runlist) (LaunchResult, error)
	PauseApp([]string) (LaunchResult, error)
}

type node struct {
	service *cocaine.Service
}

func (n *node) AppList() (appList []string, err error) {
	r := <-n.service.Call("list")
	if r.Err() != nil {
		return nil, r.Err()
	}
	r.Extract(&appList)
	return
}

func (n *node) StartApp(runlist Runlist) (l LaunchResult, err error) {
	r := <-n.service.Call("start_app", runlist)
	err = r.Err()
	if err != nil {
		return
	}
	r.Extract(&l)
	return
}

func (n *node) PauseApp(applist []string) (l LaunchResult, err error) {
	r := <-n.service.Call("pause_app", applist)
	err = r.Err()
	if err != nil {
		return
	}
	r.Extract(&l)
	return
}

func NewNode(endpoint string) (n Node, err error) {
	s, err := cocaine.NewService("node", endpoint)
	if err != nil {
		return
	}
	n = &node{s}
	return
}
