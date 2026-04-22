package utils

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/spf13/viper"
)

func TracksDuration(tracks ...lavalink.Track) string {
	totalDuration := lavalink.Duration(0)
	for _, track := range tracks {
		totalDuration += track.Info.Length
	}

	return FormatDuration(totalDuration)
}

func FormatDuration(duration lavalink.Duration) string {
	if duration < lavalink.Duration(1) {
		return "00:00"
	}

	hours := duration.Hours()
	minutes := duration.MinutesPart()
	seconds := duration.SecondsPart()

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func NewBaseEmbed() *discord.Embed {
	return new(discord.NewEmbed().WithColor(viper.GetInt("color")))
}

func JoinCustomID(args ...string) string {
	return strings.Join(args, "/")
}

func GetDeclensionWord(number int, forms [3]string) string {
	number %= 100
	if number > 19 {
		number %= 10
	}

	if number == 1 {
		return forms[0]
	}

	if number >= 2 && number <= 4 {
		return forms[1]
	}

	return forms[2]
}

func PtrEqual[T comparable](a, b *T) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func Ternary[T any](exp bool, ifCond T, elseCond T) T {
	if exp {
		return ifCond
	}
	return elseCond
}
