package components

import (
	"errors"
	"fmt"
)

var componentsMap = make(map[string]Component)

func Register(name string, component Component) {
	if component == nil {
		panic(fmt.Sprintf("Component %s does not exist.", name))
	}
	_, registered := componentsMap[name]
	if registered {
		panic(fmt.Sprintf("Component %s already registered. Ignoring.", name))
	}

	componentsMap[name] = component
}

func GetComponent(name string) (Component, error) {
	component, ok := componentsMap[name]
	if !ok {
		return nil, errors.New("Component factory" + name + " not found")
	}

	return component.(Component), nil
}
