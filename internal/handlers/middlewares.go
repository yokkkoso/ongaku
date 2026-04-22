package handlers

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"gitlab.com/yokkkoso/musicbot/internal/utils/exptime"
)

func HasExpiredTime(next handler.Handler) handler.Handler {
	return func(e *handler.InteractionEvent) error {
		expTime, ok := e.Vars["expTime"]
		if ok {
			if exptime.IsExpired(expTime) {
				switch e.Type() {
				case discord.InteractionTypeComponent:
					{
						componentInteraction := e.Interaction.(discord.ComponentInteraction)
						_ = e.Client().Rest.DeleteMessage(
							componentInteraction.Channel().ID(),
							componentInteraction.Message.ID,
						)
					}
				default:
					{
					}
				}

				return nil
			}
		}

		return next(e)
	}
}

func IsByExecutor(next handler.Handler) handler.Handler {
	return func(e *handler.InteractionEvent) error {
		executorID, ok := e.Vars["executorID"]
		if ok {
			if executorID != e.User().ID.String() {
				return nil
			}
		}

		return next(e)
	}
}
