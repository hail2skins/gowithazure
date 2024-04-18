package utility

import "log"

func HandleError(err error) {
	if err != nil {
		log.Println(err.Error()) // Log the error and continue execution
	}
}
