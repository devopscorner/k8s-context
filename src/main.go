package main

import (
	"fmt"

	"github.com/devopscorner/k8s-context/src/features"
	"github.com/muesli/termenv"
)

func main() {
	logoStyle := termenv.Style{}.Foreground(termenv.ANSIGreen)
	authorStyle := termenv.Style{}.Foreground(termenv.ANSIBlue)
	appNameStyle := termenv.Style{}.Foreground(termenv.ANSIWhite).Bold()

	fmt.Print(logoStyle.Styled(features.Logo))
	fmt.Println(authorStyle.Styled(features.Author))
	fmt.Println("===================================")
	fmt.Println("[[ ", appNameStyle.Styled(features.AppName), " ]] -", features.VERSION)
	fmt.Println("===================================")
	features.GetCommands()
}
