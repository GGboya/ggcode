-- GGCode 数据库迁移脚本
-- 添加 UserStudyPlan 表的唯一性约束

-- 1. 检查并处理重复数据
-- 删除重复的学习计划（保留最新的）
DELETE t1 FROM user_study_plans t1
INNER JOIN user_study_plans t2 
WHERE t1.user_id = t2.user_id 
  AND t1.question_bank_id = t2.question_bank_id 
  AND t1.id < t2.id;

-- 2. 添加唯一索引
-- 检查索引是否已存在，如果不存在则创建
SELECT COUNT(*) as index_exists 
FROM INFORMATION_SCHEMA.STATISTICS 
WHERE table_schema = DATABASE() 
  AND table_name = 'user_study_plans' 
  AND index_name = 'idx_user_questionbank';

-- 如果上面的查询返回0，则执行下面的语句
ALTER TABLE user_study_plans 
ADD UNIQUE INDEX idx_user_questionbank (user_id, question_bank_id);

-- 3. 验证索引创建成功
SHOW INDEX FROM user_study_plans WHERE Key_name = 'idx_user_questionbank'; 