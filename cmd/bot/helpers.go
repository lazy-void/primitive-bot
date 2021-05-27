package main

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/lazy-void/primitive-bot/pkg/menu"

	"github.com/lazy-void/primitive-bot/pkg/primitive"
	"github.com/lazy-void/primitive-bot/pkg/tg"
)

func (app *application) serverError(chatID int64, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())

	err = app.errorLog.Output(2, trace)
	if err != nil {
		app.errorLog.Print(err)
	}

	_, err = app.bot.SendMessage(chatID,
		app.printer.Sprintf("Something gone wrong! Please, try again in a few minutes."))
	if err != nil {
		app.errorLog.Print(err)
	}
}

func (app *application) createStatusMessage(c primitive.Config, position int) string {
	return app.printer.Sprintf(
		"%d place in the queue.\n\nShapes: %s\nSteps: %d\nRepetitions: %d\nAlpha-channel: %d\nExtension: %s\nSize: %#v",
		position, strings.ToLower(menu.ShapeNames[c.Shape]), c.Iterations, c.Repeat, c.Alpha, c.Extension, c.OutputSize,
	)
}

func (app *application) getInputFromUser(
	chatID, menuMessageID int64,
	min, max int,
	in <-chan tg.Message,
	out chan<- int,
) {
	err := app.bot.EditMessageText(chatID, menuMessageID,
		app.printer.Sprintf("Enter number between %#v and %#v:", min, max))
	if err != nil {
		app.serverError(chatID, err)
		return
	}

	for {
		userMsg := <-in
		if err := app.bot.DeleteMessage(userMsg.Chat.ID, userMsg.MessageID); err != nil {
			app.serverError(chatID, err)
			return
		}

		userInput, err := strconv.Atoi(userMsg.Text)
		// correct input
		if err == nil && userInput >= min && userInput <= max {
			out <- userInput
			close(out)
			return
		}

		// incorrect input
		err = app.bot.EditMessageText(chatID, menuMessageID,
			app.printer.Sprintf("Incorrect value!\nEnter number between %#v and %#v:", min, max))
		if err != nil {
			if strings.Contains(err.Error(), "400") {
				// 400 error: message is not modified
				// and we don't care in this case
				continue
			}
			app.serverError(chatID, err)
			return
		}
	}
}

func (app *application) showMenuView(
	chatID, messageID int64,
	view menu.View,
) {
	err := app.bot.EditMessageText(chatID, messageID, view.Text, view.Keyboard)
	if err != nil {
		if strings.Contains(err.Error(), "400") {
			// 400 error: message is not modified
			// and we don't care in this case
			return
		}
		app.serverError(chatID, err)
	}
}

// match reports whether path matches ^pattern$, and if it matches,
// assigns any capture groups to the *string or *int vars.
func match(path, pattern string, vars ...interface{}) bool {
	regex := mustCompileCached(fmt.Sprintf("^%s$", pattern))
	matches := regex.FindStringSubmatch(path)
	if len(matches) == 0 {
		return false
	}

	for i, match := range matches[1:] {
		switch p := vars[i].(type) {
		case *string:
			*p = match
		case *int:
			n, err := strconv.Atoi(match)
			if err != nil {
				return false
			}
			*p = n
		default:
			return false
		}
	}
	return true
}

var (
	regexen = make(map[string]*regexp.Regexp)
	relock  = sync.Mutex{}
)

func mustCompileCached(pattern string) *regexp.Regexp {
	relock.Lock()
	defer relock.Unlock()

	regex := regexen[pattern]
	if regex == nil {
		regex = regexp.MustCompile(pattern)
		regexen[pattern] = regex
	}
	return regex
}
