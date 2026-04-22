package progress_bar

import "strings"

const (
	totalBlocks    = 20
	filledBlock    = "▬"
	emptyBlock     = "▬"
	positionMarker = "🔘"
)

func ProgressBar(current, total int64) string {
	position := int(float64(current) / float64(total) * float64(totalBlocks))

	if position < 0 {
		position = 0
	} else if position > totalBlocks {
		position = totalBlocks
	}

	var result strings.Builder
	result.WriteString("[")

	if position > 0 {
		result.WriteString(strings.Repeat(filledBlock, position-1))
	}

	result.WriteString(positionMarker)

	remainingBlocks := totalBlocks - position
	if remainingBlocks > 0 {
		result.WriteString(strings.Repeat(emptyBlock, remainingBlocks))
	}

	result.WriteString("]")

	return result.String()
}
