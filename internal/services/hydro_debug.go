package services

import (
	"fmt"
	"log"
	"time"
)

// DebugHydroJudge 调试Hydro评测系统
func DebugHydroJudge(service HydroJudgeService) {
	log.Println("=== Hydro评测系统调试信息 ===")

	// 1. 检查队列状态
	status := service.GetQueueStatus()
	log.Printf("队列状态: 待处理=%d, 评测中=%d, 已完成=%d",
		status.PendingCount, status.JudgingCount, status.TotalJudged)

	// 2. 提交测试代码
	testSubmission := &JudgeSubmission{
		ID:         uint(time.Now().Unix()),
		UserID:     999, // 测试用户ID
		LevelID:    1,   // 测试关卡ID
		Code:       `#include <iostream>\nusing namespace std;\nint main() {\n    int a, b;\n    cin >> a >> b;\n    cout << a + b << endl;\n    return 0;\n}`,
		Language:   "cpp",
		SubmitTime: 0,
		Priority:   1,
	}

	log.Printf("提交测试代码: ID=%d", testSubmission.ID)
	result, err := service.SubmitForJudge(testSubmission)
	if err != nil {
		log.Printf("提交失败: %v", err)
		return
	}

	log.Printf("初始结果: Status=%s, SubmissionID=%d", result.Status, result.SubmissionID)

	// 3. 轮询等待结果
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)

		currentResult, err := service.GetJudgeResult(testSubmission.ID)
		if err != nil {
			log.Printf("获取结果失败: %v", err)
			continue
		}

		log.Printf("第%d次查询: Status=%s, Score=%d/%d, Time=%dms, Error=%s",
			i+1, currentResult.Status, currentResult.Score, currentResult.MaxScore,
			currentResult.TimeUsed, currentResult.Error)

		if currentResult.Status != StatusPending && currentResult.Status != StatusJudging {
			log.Printf("评测完成: 最终状态=%s", currentResult.Status)
			if len(currentResult.TestCases) > 0 {
				for j, testCase := range currentResult.TestCases {
					log.Printf("测试点%d: Status=%s, Score=%d/%d, Time=%dms",
						j+1, testCase.Status, testCase.Score, testCase.MaxScore, testCase.TimeUsed)
				}
			}
			break
		}

		if i == 9 {
			log.Printf("评测超时，最终状态: %s", currentResult.Status)
		}
	}

	// 4. 再次检查队列状态
	finalStatus := service.GetQueueStatus()
	log.Printf("最终队列状态: 待处理=%d, 评测中=%d, 已完成=%d",
		finalStatus.PendingCount, finalStatus.JudgingCount, finalStatus.TotalJudged)

	log.Println("=== 调试完成 ===")
}

// CheckWorkerStatus 检查工作者状态
func CheckWorkerStatus(service *hydroJudgeService) {
	log.Println("=== 工作者状态检查 ===")
	log.Printf("服务运行状态: %v", service.running)
	log.Printf("最大工作者数: %d", service.maxWorkers)
	log.Printf("实际工作者数: %d", len(service.workers))
	log.Printf("任务队列长度: %d", len(service.taskQueue))
	log.Printf("任务队列容量: %d", cap(service.taskQueue))

	// 检查语言配置
	log.Println("支持的语言:")
	for lang, config := range service.languageConfigs {
		log.Printf("  %s: %s", lang, config.Name)
	}

	log.Println("=== 检查完成 ===")
}

// SimulateLoad 模拟负载测试
func SimulateLoad(service HydroJudgeService, numTasks int) {
	log.Printf("=== 开始负载测试: %d个任务 ===", numTasks)

	startTime := time.Now()

	// 提交多个任务
	submissionIDs := make([]uint, numTasks)
	for i := 0; i < numTasks; i++ {
		submission := &JudgeSubmission{
			ID:         uint(time.Now().UnixNano()/1000000) + uint(i), // 使用纳秒确保唯一性
			UserID:     uint(1000 + i),
			LevelID:    1,
			Code:       fmt.Sprintf(`#include <iostream>\nusing namespace std;\nint main() {\n    cout << %d << endl;\n    return 0;\n}`, i),
			Language:   "cpp",
			SubmitTime: 0,
			Priority:   1,
		}

		_, err := service.SubmitForJudge(submission)
		if err != nil {
			log.Printf("任务%d提交失败: %v", i, err)
			continue
		}
		submissionIDs[i] = submission.ID

		// 小延迟，避免过快提交
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("所有任务已提交，等待完成...")

	// 等待所有任务完成
	completed := 0
	for completed < numTasks {
		time.Sleep(500 * time.Millisecond)

		currentCompleted := 0
		for _, id := range submissionIDs {
			if id == 0 {
				continue
			}

			result, err := service.GetJudgeResult(id)
			if err != nil {
				continue
			}

			if result.Status != StatusPending && result.Status != StatusJudging {
				currentCompleted++
			}
		}

		if currentCompleted != completed {
			completed = currentCompleted
			log.Printf("进度: %d/%d 任务完成", completed, numTasks)
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	log.Printf("=== 负载测试完成 ===")
	log.Printf("总用时: %v", duration)
	log.Printf("平均每任务: %v", duration/time.Duration(numTasks))
	log.Printf("并发处理能力: %.2f tasks/second", float64(numTasks)/duration.Seconds())

	// 最终状态
	finalStatus := service.GetQueueStatus()
	log.Printf("最终队列状态: 待处理=%d, 评测中=%d, 已完成=%d",
		finalStatus.PendingCount, finalStatus.JudgingCount, finalStatus.TotalJudged)
}
