package ssh

import (
	"golang.org/x/crypto/ssh"
	"log"
)

// An implementation of ssh.KeyboardInteractiveChallenge that simply sends
// back the password for all questions. The questions are logged.
func PasswordKeyboardInteractive(password string) ssh.KeyboardInteractiveChallenge {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		log.Printf("Keyboard interactive challenge: ")
		log.Printf("-- User: %s", user)
		log.Printf("-- Instructions: %s", instruction)
		for i, question := range questions {
			log.Printf("-- Question %d: %s", i+1, question)
		}

		// Just send the password back for all questions
		answers := make([]string, len(questions))
		for i := range answers {
			answers[i] = string(password)
		}

		return answers, nil
	}
}
