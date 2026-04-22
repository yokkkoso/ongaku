package pagination

import (
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
	"gitlab.com/yokkkoso/musicbot/internal/utils"
)

var buttonsEmoji = map[PageAction]string{
	FirstPageAction: "⏪",
	PrevPageAction:  "◀",
	NextPageAction:  "▶",
	LastPageAction:  "⏩",
}

type PageAction string

const (
	FirstPageAction PageAction = "F"
	PrevPageAction  PageAction = "P"
	NextPageAction  PageAction = "N"
	LastPageAction  PageAction = "L"
)

var buttonIndexMap = map[PageAction]map[bool]int{
	FirstPageAction: {true: 0},
	PrevPageAction:  {true: 1, false: 0},
	NextPageAction:  {true: 2, false: 1},
	LastPageAction:  {true: 3},
}

func createButton(
	action PageAction,
	suffix, expTime string,
	executorID snowflake.ID,
	currentPage, totalPages int,
) discord.ButtonComponent {
	return discord.ButtonComponent{
		Style: discord.ButtonStyleSecondary,
		CustomID: utils.JoinCustomID(
			"/pagination",
			expTime,
			executorID.String(),
			string(action),
			strconv.Itoa(currentPage),
			strconv.Itoa(totalPages),
			suffix,
		),
		Emoji: &discord.ComponentEmoji{
			Name: buttonsEmoji[action],
		},
	}
}

func GeneratePageButtons(
	suffix string,
	expTime string,
	executorID snowflake.ID,
	currentPage,
	totalPages int,
) discord.ActionRowComponent {
	var buttons []discord.InteractiveComponent
	hasMultiplePages := totalPages > 2

	if hasMultiplePages {
		buttons = []discord.InteractiveComponent{
			createButton(FirstPageAction, suffix, expTime, executorID, currentPage, totalPages),
			createButton(PrevPageAction, suffix, expTime, executorID, currentPage, totalPages),
			createButton(NextPageAction, suffix, expTime, executorID, currentPage, totalPages),
			createButton(LastPageAction, suffix, expTime, executorID, currentPage, totalPages),
		}
	} else {
		buttons = []discord.InteractiveComponent{
			createButton(PrevPageAction, suffix, expTime, executorID, currentPage, totalPages),
			createButton(NextPageAction, suffix, expTime, executorID, currentPage, totalPages),
		}
	}

	actionRow := discord.NewActionRow(buttons...)

	if currentPage == 1 {
		EditButtonsAvailability(actionRow, []PageAction{FirstPageAction, PrevPageAction}, totalPages, true)
	}

	if currentPage == totalPages {
		EditButtonsAvailability(actionRow, []PageAction{NextPageAction, LastPageAction}, totalPages, true)
	}

	return actionRow
}

func EditButtonsAvailability(
	actionRow discord.ActionRowComponent,
	buttons []PageAction,
	totalPages int,
	isDisable bool,
) discord.ActionRowComponent {
	hasMultiplePages := totalPages > 2

	for _, button := range buttons {
		if index, ok := buttonIndexMap[button][hasMultiplePages]; ok {
			actionRow.Components[index] = actionRow.Components[index].(discord.ButtonComponent).WithDisabled(isDisable)
		}
	}

	return actionRow
}
