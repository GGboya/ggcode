package events

type UserCompletedQuestionEvent struct {
	UserID     uint
	QuestionID uint
}

type EventBus struct {
	UserCompletedQuestionChan chan UserCompletedQuestionEvent
}

func NewEventBus() *EventBus {
	return &EventBus{
		UserCompletedQuestionChan: make(chan UserCompletedQuestionEvent, 100),
	}
}
