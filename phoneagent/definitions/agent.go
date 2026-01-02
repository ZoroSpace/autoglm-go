package definitions

import (
	"fmt"
	"time"

	"autoglm-go/constants"
)

type AgentConfig struct {
	MaxSteps int
	DeviceID string
	Lang     string
	WdaUrl   string // ios only
}

func (c *AgentConfig) GetSystemPrompt() string {
	today := time.Now()

	if c.Lang == "en" {
		return fmt.Sprintf(constants.DefaultEnSystemPrompt, today.Format("2006-01-02, Monday"))
	}

	weekdayNames := []string{"星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}
	weekday := weekdayNames[today.Weekday()]
	return fmt.Sprintf(constants.DefaultCnSystemPrompt, today.Format("2006年01月02日")+" "+weekday)
}
