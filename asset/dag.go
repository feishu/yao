package asset

import (
	"fmt"
	"sort"
)

// TopologicalSort 对资产定义进行拓扑依赖排序
// 如果 A DependsOn B，则 B 会排在 A 前面
func TopologicalSort(defs []Definition) ([]Definition, error) {
	if len(defs) <= 1 {
		return defs, nil
	}

	defMap := make(map[string]Definition, len(defs))
	inDegree := make(map[string]int, len(defs))
	dependents := make(map[string][]string, len(defs)) // key 被哪些依赖：B -> [A]

	for _, def := range defs {
		defMap[def.Name] = def
		inDegree[def.Name] = 0
	}

	for _, def := range defs {
		for _, dep := range def.DependsOn {
			// 只在当前需要加载的资产集合中构建依赖图
			if _, exists := defMap[dep]; exists {
				inDegree[def.Name]++
				dependents[dep] = append(dependents[dep], def.Name)
			}
		}
	}

	// 零入度队列，为了保证确定性顺序，按 Name 排序
	queue := make([]string, 0)
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	result := make([]Definition, 0, len(defs))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		result = append(result, defMap[curr])

		// 获取依赖当前节点的所有节点并按字典序排序，确保稳定性
		nextNodes := dependents[curr]
		sort.Strings(nextNodes)
		for _, next := range nextNodes {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
		sort.Strings(queue)
	}

	if len(result) != len(defs) {
		// 存在环
		unresolved := make([]string, 0)
		for name, deg := range inDegree {
			if deg > 0 {
				unresolved = append(unresolved, fmt.Sprintf("%s(indegree=%d)", name, deg))
			}
		}
		sort.Strings(unresolved)
		return nil, fmt.Errorf("circular asset dependency detected among: %v", unresolved)
	}

	return result, nil
}

// TopologicalSortStages 将资产定义按拓扑依赖划分为阶段波次（Waves）
// 同一个阶段（Stage）内部的资产互不依赖，可以安全地并发加载
func TopologicalSortStages(defs []Definition) ([][]Definition, error) {
	if len(defs) == 0 {
		return [][]Definition{}, nil
	}
	if len(defs) == 1 {
		return [][]Definition{{defs[0]}}, nil
	}

	defMap := make(map[string]Definition, len(defs))
	inDegree := make(map[string]int, len(defs))
	dependents := make(map[string][]string, len(defs))

	for _, def := range defs {
		defMap[def.Name] = def
		inDegree[def.Name] = 0
	}

	for _, def := range defs {
		for _, dep := range def.DependsOn {
			if _, exists := defMap[dep]; exists {
				inDegree[def.Name]++
				dependents[dep] = append(dependents[dep], def.Name)
			}
		}
	}

	// 收集第一批入度为 0 的节点
	currentWave := make([]string, 0)
	for name, deg := range inDegree {
		if deg == 0 {
			currentWave = append(currentWave, name)
		}
	}
	sort.Strings(currentWave)

	stages := make([][]Definition, 0)
	totalProcessed := 0

	for len(currentWave) > 0 {
		stage := make([]Definition, 0, len(currentWave))
		nextWave := make([]string, 0)

		for _, name := range currentWave {
			stage = append(stage, defMap[name])
			totalProcessed++

			nextNodes := dependents[name]
			for _, next := range nextNodes {
				inDegree[next]--
				if inDegree[next] == 0 {
					nextWave = append(nextWave, next)
				}
			}
		}

		sort.Strings(nextWave)
		stages = append(stages, stage)
		currentWave = nextWave
	}

	if totalProcessed != len(defs) {
		unresolved := make([]string, 0)
		for name, deg := range inDegree {
			if deg > 0 {
				unresolved = append(unresolved, fmt.Sprintf("%s(indegree=%d)", name, deg))
			}
		}
		sort.Strings(unresolved)
		return nil, fmt.Errorf("circular asset dependency detected among: %v", unresolved)
	}

	return stages, nil
}
