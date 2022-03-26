package controller

type eventType string

const (
	addExpression    eventType = "addExpression"
	updateExpression eventType = "updateExpression"
)

type event struct {
	eventType      eventType
	oldObj, newObj interface{}
}
