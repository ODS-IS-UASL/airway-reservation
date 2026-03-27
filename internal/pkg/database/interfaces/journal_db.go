package interfaces

type JournalEvent struct {
	ID        string `json:"id"`
	EventName string `json:"eventName"`
	TableName string `json:"tableName"`
	Payload   string `json:"payload"`
	Timestamp string `json:"timestamp"`
	TopicName string `json:"topicName"`
}

type JournalDBIF interface {
	AddEvent(event *JournalEvent, tableName string) error
	AddEventsTransact(events []*JournalEvent, tableName string) error
}
