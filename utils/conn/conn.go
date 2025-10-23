package conn

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	config "github.com/yaoapp/yao/connector"
)

/*
*
// ProcessConnector 获取 Connector 实例
// @param connname string 连接器名称
// @return Connector 连接器实例
*/
func ProcessSelectConnector(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	connname := process.ArgsString(0)
	if connname == "" {
		exception.New("Connector name is required", 400).Throw()
	}
	conn, err := connector.Select(connname)
	if err != nil {
		c, err := config.SelectConfig(connname)
		if err != nil {
			exception.New("Get Connector error: %s", 400, err.Error()).Throw()
		}

		return map[string]interface{}{
			"id":      c.ID(),
			"name":    c.Name(),
			"label":   c.Label(),
			"version": c.Version(),
			"type":    c.Type(),
			"options": c.Options(),
		}
	}

	return conn
}
