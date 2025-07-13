package bot

import "fmt"

type BotError struct {
	Msg string
	Err error
}

func (e *BotError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *BotError) Unwrap() error {
	return e.Err
}

func NewBotError(msg string, err error) *BotError {
	return &BotError{Msg: msg, Err: err}
}
