package services

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SimpleContainer 简化容器
type SimpleContainer struct {
	ID       string
	InUse    bool
	LastUsed time.Time
	mutex    sync.Mutex
}

// SimpleContainerPool 简化容器池
type SimpleContainerPool struct {
	containers []*SimpleContainer
	mutex      sync.RWMutex
	image      string
	running    bool
}

// NewSimpleContainerPool 创建简化容器池
func NewSimpleContainerPool() *SimpleContainerPool {
	pool := &SimpleContainerPool{
		containers: make([]*SimpleContainer, 0),
		image:      "ggcode-judge:stable",
		running:    true,
	}

	// 启动容器池
	if err := pool.startPool(); err != nil {
		log.Printf("容器池启动失败: %v", err)
		return nil
	}

	return pool
}

// startPool 启动容器池
func (p *SimpleContainerPool) startPool() error {
	// 创建统一的容器池，不区分语言
	// 2H2G服务器创建6个容器比较合适
	containerCount := 6

	for i := 0; i < containerCount; i++ {
		containerID, err := p.createContainer()
		if err != nil {
			log.Printf("创建容器失败: %v", err)
			continue
		}

		container := &SimpleContainer{
			ID:       containerID,
			InUse:    false,
			LastUsed: time.Now(),
		}

		p.containers = append(p.containers, container)
	}

	// 启动监控
	go p.monitor()

	return nil
}

// createContainer 创建容器
func (p *SimpleContainerPool) createContainer() (string, error) {
	// 使用命令行Docker创建容器
	cmd := exec.Command("docker", "run", "-d",
		"--memory=128m",
		"--cpus=0.2",
		"--network=none",
		"--label", "ggcode.service=judge",
		"--label", "ggcode.pool=true",
		p.image)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("创建容器失败: %v", err)
	}

	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

// GetContainer 获取可用容器
func (p *SimpleContainerPool) GetContainer(language string) (*SimpleContainer, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 找任意空闲容器
	for _, container := range p.containers {
		container.mutex.Lock()
		if !container.InUse {
			container.InUse = true
			container.LastUsed = time.Now()
			container.mutex.Unlock()
			log.Printf("分配容器: %s (语言=%s)", container.ID[:12], language)
			return container, nil
		}
		container.mutex.Unlock()
	}

	return nil, fmt.Errorf("没有可用的容器")
}

// ReleaseContainer 释放容器
func (p *SimpleContainerPool) ReleaseContainer(containerID string) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, container := range p.containers {
		if container.ID == containerID {
			container.mutex.Lock()
			container.InUse = false
			container.LastUsed = time.Now()
			container.mutex.Unlock()

			// 清理容器
			p.cleanContainer(containerID)

			log.Printf("释放容器: %s", containerID[:12])
			return nil
		}
	}

	return fmt.Errorf("容器不存在: %s", containerID)
}

// cleanContainer 清理容器
func (p *SimpleContainerPool) cleanContainer(containerID string) {
	cmd := exec.Command("docker", "exec", containerID,
		"bash", "-c", "rm -rf /opt/judge/* && mkdir -p /opt/judge")
	cmd.Run() // 忽略错误
}

// Shutdown 关闭容器池
func (p *SimpleContainerPool) Shutdown() error {
	p.running = false

	p.mutex.RLock()
	containers := make([]*SimpleContainer, len(p.containers))
	copy(containers, p.containers)
	p.mutex.RUnlock()

	if len(containers) == 0 {
		return nil
	}

	log.Printf("正在并行停止 %d 个容器...", len(containers))

	// 第一阶段：并行发送停止信号给所有容器
	var containerIDs []string
	for _, container := range containers {
		containerIDs = append(containerIDs, container.ID)
	}

	// 使用单个docker命令停止所有容器，更快
	stopCmd := append([]string{"stop", "--time", "5"}, containerIDs...)
	if err := exec.Command("docker", stopCmd...).Run(); err != nil {
		log.Printf("批量停止容器失败，尝试单独停止: %v", err)
		// 如果批量停止失败，单独停止每个容器
		for _, containerID := range containerIDs {
			exec.Command("docker", "kill", containerID).Run()
		}
	}

	// 第二阶段：并行删除所有容器
	rmCmd := append([]string{"rm"}, containerIDs...)
	if err := exec.Command("docker", rmCmd...).Run(); err != nil {
		log.Printf("批量删除容器失败，尝试单独删除: %v", err)
		// 如果批量删除失败，单独删除每个容器
		for _, containerID := range containerIDs {
			exec.Command("docker", "rm", containerID).Run()
		}
	}

	log.Printf("所有 %d 个容器已停止并删除", len(containers))
	return nil
}

// monitor 监控容器
func (p *SimpleContainerPool) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for p.running {
		<-ticker.C
		p.printStats()
	}
}

// printStats 打印统计
func (p *SimpleContainerPool) printStats() {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	total := len(p.containers)
	inUse := 0

	for _, container := range p.containers {
		// 读取bool值不需要加锁，Go中bool读写是原子的
		if container.InUse {
			inUse++
		}
	}

	log.Printf("容器池状态: 总数=%d, 使用中=%d, 空闲=%d", total, inUse, total-inUse)
}

// GetStats 获取统计信息
func (p *SimpleContainerPool) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	total := len(p.containers)
	inUse := 0

	for _, container := range p.containers {
		// 读取bool值不需要加锁，Go中bool读写是原子的
		if container.InUse {
			inUse++
		}
	}

	return map[string]interface{}{
		"total_containers": total,
		"busy_containers":  inUse,
		"idle_containers":  total - inUse,
	}
}
